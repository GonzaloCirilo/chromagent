package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/GonzaloCirilo/chromagent/internal/chroma"
	"github.com/GonzaloCirilo/chromagent/internal/config"
	"github.com/GonzaloCirilo/chromagent/internal/service"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServe()
		return
	}
	runClient()
}

// runServe starts the long-running daemon: initializes the Chroma SDK and
// serves effect commands over a Unix socket until all agent sessions end.
func runServe() {
	cfg := config.Load()

	client, err := chroma.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[chromagent] Chroma SDK init failed: %v\n", err)
		os.Exit(1)
	}
	// No defer client.Close() — service.doShutdown owns the lifecycle.

	srv := service.New(cfg, client)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "[chromagent] Server error: %v\n", err)
		os.Exit(1)
	}
}

// runClient is the hook entrypoint: reads stdin, ensures the daemon is running,
// and forwards the event. Always exits 0 to never block the agent's workflow.
func runClient() {
	var raw json.RawMessage
	if err := json.NewDecoder(os.Stdin).Decode(&raw); err != nil {
		fmt.Fprintf(os.Stderr, "[chromagent] Failed to read stdin: %v\n", err)
		os.Exit(0)
	}

	socketPath := service.SocketPath()

	if err := ensureServiceRunning(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "[chromagent] Could not start service: %v\n", err)
		os.Exit(0)
	}

	if err := forwardEvent(socketPath, raw); err != nil {
		fmt.Fprintf(os.Stderr, "[chromagent] Forward failed: %v\n", err)
	}

	os.Exit(0)
}

// ensureServiceRunning checks if the daemon is listening on the socket, and
// spawns it in the background if not, polling until it becomes ready.
func ensureServiceRunning(socketPath string) error {
	if socketReady(socketPath) {
		return nil
	}

	if err := spawnDaemon(); err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if socketReady(socketPath) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("service did not become ready within 10 seconds")
}

// socketReady returns true if the Unix socket accepts connections.
func socketReady(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// detachedProcess is the Windows DETACHED_PROCESS creation flag (0x00000008).
// It prevents the child from inheriting the parent's console window.
const detachedProcess = 0x00000008

// spawnDaemon starts `chromagent serve` as a detached background process.
func spawnDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	cmd := exec.Command(exe, "serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: detachedProcess,
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	// Disown the process — we don't wait for it.
	return cmd.Process.Release()
}

// forwardEvent sends the raw event JSON to the running service via the Unix socket.
// It waits for the response so the effect completes before the hook process exits.
func forwardEvent(socketPath string, raw json.RawMessage) error {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   35 * time.Second, // generous: longest effect (Pulse) ~1.6s
	}

	resp, err := client.Post("http://chromagent/event", "application/json", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("post event: %w", err)
	}
	resp.Body.Close()
	return nil
}
