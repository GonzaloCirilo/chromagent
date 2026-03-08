package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GonzaloCirilo/chromagent/internal/agents"
	"github.com/GonzaloCirilo/chromagent/internal/chroma"
	"github.com/GonzaloCirilo/chromagent/internal/config"

	// Register adapters via init().
	_ "github.com/GonzaloCirilo/chromagent/internal/agents/claude"
	_ "github.com/GonzaloCirilo/chromagent/internal/agents/cursor"
)

// SocketPath returns the Unix socket path used for IPC.
func SocketPath() string {
	return filepath.Join(os.TempDir(), "chromagent.sock")
}

// Server is the long-running daemon that holds the Chroma SDK session and
// serves effect commands from short-lived hook processes via a Unix socket.
type Server struct {
	cfg        *config.Config
	chroma     *chroma.Client
	httpServer *http.Server
	socketPath string

	mu             sync.Mutex
	activeSessions map[string]struct{}
	shutdownTimer  *time.Timer
}

// New creates a Server. The caller owns chromaClient — the server will call
// Close() on it when all sessions end.
func New(cfg *config.Config, chromaClient *chroma.Client) *Server {
	s := &Server{
		cfg:            cfg,
		chroma:         chromaClient,
		socketPath:     SocketPath(),
		activeSessions: make(map[string]struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/event", s.handleEvent)

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second, // Pulse effect can run up to ~1.6s
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// ListenAndServe starts the Unix socket listener and blocks until the server
// shuts down. If another daemon is already running it exits immediately.
func (s *Server) ListenAndServe() error {
	// If the socket exists and a daemon is responding, exit silently —
	// another instance is already running and will handle events.
	if conn, err := net.DialTimeout("unix", s.socketPath, 300*time.Millisecond); err == nil {
		conn.Close()
		log.Printf("[chromagent] Daemon already running on %s — exiting.", s.socketPath)
		return nil
	}

	// No live daemon. Remove any stale socket file from a previous crash.
	os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on unix socket %s: %w", s.socketPath, err)
	}

	log.Printf("[chromagent] Service listening on %s", s.socketPath)

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleHealth responds to GET /health with 200 OK.
// Hook processes poll this to confirm the service is ready.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

// handleEvent accepts POST /event with a raw hook JSON body,
// dispatches the effect, and updates session tracking.
// Always responds 200 to avoid blocking the agent workflow.
func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	var raw json.RawMessage = body
	if !json.Valid(raw) {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	adapter := agents.DetectAdapter(raw)
	if adapter == nil {
		log.Printf("[chromagent] No adapter matched; ignoring event")
		w.WriteHeader(http.StatusOK)
		return
	}

	event, err := adapter.ParseEvent(raw)
	if err != nil {
		log.Printf("[chromagent] ParseEvent error: %v", err)
		w.WriteHeader(http.StatusOK) // best-effort
		return
	}

	sessionID := extractSessionID(raw)
	s.updateSessions(event, sessionID)
	s.dispatchEffect(event)

	w.WriteHeader(http.StatusOK)
}

// extractSessionID pulls the session identifier from a raw hook payload.
// Claude Code uses "session_id"; Cursor uses "conversation_id".
func extractSessionID(raw json.RawMessage) string {
	var probe struct {
		SessionID      string `json:"session_id"`
		ConversationID string `json:"conversation_id"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ""
	}
	if probe.SessionID != "" {
		return probe.SessionID
	}
	return probe.ConversationID
}

// updateSessions maintains the active session map and manages the shutdown timer.
//
// Edge case — service starts mid-session: if the daemon was auto-started after
// a SessionStart already fired, we learn about the session from the first
// subsequent event (default branch below). This prevents premature shutdown.
func (s *Server) updateSessions(event agents.AgentEvent, sessionID string) {
	if sessionID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch event {
	case agents.EventSessionStart:
		// New session — cancel any pending shutdown.
		if s.shutdownTimer != nil {
			s.shutdownTimer.Stop()
			s.shutdownTimer = nil
			log.Printf("[chromagent] Cancelled shutdown; new session: %s", sessionID)
		}
		s.activeSessions[sessionID] = struct{}{}
		log.Printf("[chromagent] Session started: %s (active: %d)", sessionID, len(s.activeSessions))

	case agents.EventSessionEnd:
		delete(s.activeSessions, sessionID)
		log.Printf("[chromagent] Session ended: %s (remaining: %d)", sessionID, len(s.activeSessions))
		if len(s.activeSessions) == 0 && s.shutdownTimer == nil {
			log.Printf("[chromagent] No active sessions; releasing SDK in 3s")
			s.shutdownTimer = time.AfterFunc(3*time.Second, s.doShutdown)
		}

	default:
		// Any mid-session event implicitly registers the session as active.
		// This handles the case where the daemon started after SessionStart fired.
		if _, known := s.activeSessions[sessionID]; !known {
			log.Printf("[chromagent] Mid-session registration: %s", sessionID)
			// Cancel any pending shutdown triggered before we learned about this session.
			if s.shutdownTimer != nil {
				s.shutdownTimer.Stop()
				s.shutdownTimer = nil
			}
			s.activeSessions[sessionID] = struct{}{}
		}
	}
}

// doShutdown is called by the shutdown timer. It re-checks that no sessions
// have arrived during the grace period before releasing the SDK.
func (s *Server) doShutdown() {
	s.mu.Lock()
	if len(s.activeSessions) > 0 {
		// A new session arrived during the grace period — stay alive.
		s.shutdownTimer = nil
		s.mu.Unlock()
		return
	}
	s.shutdownTimer = nil
	s.mu.Unlock()

	log.Printf("[chromagent] All sessions ended. Releasing Chroma SDK.")
	s.chroma.Close()
	os.Remove(s.socketPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.httpServer.Shutdown(ctx) //nolint:errcheck

	log.Printf("[chromagent] Service exiting.")
	os.Exit(0)
}

// idleColor is the steady-state color shown while the daemon is connected.
var idleColor = chroma.ColorWhite

// restoreIdle sets all devices to the idle (white) color.
func (s *Server) restoreIdle() {
	s.chroma.StaticAll(idleColor) //nolint:errcheck
}

// dispatchEffect routes a canonical event to the appropriate Chroma effect,
// using user-configurable colors from the config.
// Most effects restore the idle color after completing.
// Exceptions: SessionEnd (clears), ToolExecuting (holds until ToolCompleted),
// NeedsAttention (holds red so user notices).
func (s *Server) dispatchEffect(event agents.AgentEvent) {
	color := s.cfg.Events[event.Name()]
	r, g, b := color[0], color[1], color[2]
	packed := chroma.BGR(r, g, b)

	switch event {

	case agents.EventSessionStart:
		s.chroma.Pulse(r, g, b, 15, 1500*time.Millisecond)
		s.restoreIdle()

	case agents.EventSessionEnd:
		// Fade to dark — don't restore idle, session is ending.
		s.chroma.Pulse(r, g, b, 10, 800*time.Millisecond)
		s.chroma.ClearAll()

	case agents.EventPromptAcknowledged:
		s.chroma.Flash(packed, 300*time.Millisecond)
		s.restoreIdle()

	case agents.EventToolExecuting:
		// Hold yellow — restored when ToolCompleted fires.
		s.chroma.StaticAll(packed)

	case agents.EventToolCompleted:
		s.chroma.Flash(packed, 400*time.Millisecond)
		s.restoreIdle()

	case agents.EventTaskComplete:
		s.chroma.Pulse(r, g, b, 12, 1200*time.Millisecond)
		s.restoreIdle()

	case agents.EventSubtaskComplete:
		s.chroma.Flash(packed, 300*time.Millisecond)
		s.restoreIdle()

	case agents.EventNeedsAttention:
		// Alert flash then hold red — user must see it.
		// Restored on next event (e.g. ToolExecuting or PromptAcknowledged).
		s.chroma.AlertFlash(packed)
		s.chroma.StaticAll(packed) //nolint:errcheck

	case agents.EventWaitingForInput:
		s.chroma.Pulse(r, g, b, 10, 1000*time.Millisecond)
		s.restoreIdle()

	case agents.EventContextCompacting:
		s.chroma.Flash(packed, 500*time.Millisecond)
		s.restoreIdle()

	case agents.EventNotification:
		s.chroma.Flash(packed, 200*time.Millisecond)
		s.restoreIdle()

	case agents.EventUnknown:
		// Silently ignore unrecognized events.
	}
}
