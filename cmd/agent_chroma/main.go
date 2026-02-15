package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/GonzaloCirilo/agent-chroma/internal/agents"
	"github.com/GonzaloCirilo/agent-chroma/internal/chroma"
	"github.com/GonzaloCirilo/agent-chroma/internal/config"

	// Register adapters via init().
	_ "github.com/GonzaloCirilo/agent-chroma/internal/agents/claude"
	_ "github.com/GonzaloCirilo/agent-chroma/internal/agents/cursor"
)

// handleEvent routes a canonical AgentEvent to the appropriate Chroma effect,
// using user-configurable colors from the config.
func handleEvent(c *chroma.Client, event agents.AgentEvent, cfg *config.Config) {
	color := cfg.Events[event.Name()]
	r, g, b := color[0], color[1], color[2]
	packed := chroma.BGR(r, g, b)

	switch event {

	case agents.EventSessionStart:
		// Arc reactor boot-up — a calm pulse.
		c.Pulse(r, g, b, 15, 1500*time.Millisecond)

	case agents.EventSessionEnd:
		// Fade to dark.
		c.Pulse(r, g, b, 10, 800*time.Millisecond)
		c.ClearAll()

	case agents.EventPromptAcknowledged:
		// Subtle flash — acknowledged, processing.
		c.Flash(packed, 300*time.Millisecond)

	case agents.EventToolExecuting:
		// Static while a tool is running — "working on it."
		c.StaticAll(packed)

	case agents.EventToolCompleted:
		// Flash — tool completed successfully.
		c.Flash(packed, 400*time.Millisecond)

	case agents.EventTaskComplete:
		// Pulse — task complete. All done, sir.
		c.Pulse(r, g, b, 12, 1200*time.Millisecond)

	case agents.EventSubtaskComplete:
		c.Flash(packed, 300*time.Millisecond)

	case agents.EventNeedsAttention:
		// Alert flash — NEEDS YOUR ATTENTION.
		c.AlertFlash(packed)
		// Leave color on so you notice it.
		err := c.StaticAll(packed)
		if err != nil {
			_ = fmt.Errorf("")
		}

	case agents.EventWaitingForInput:
		// Pulse — waiting for input.
		c.Pulse(r, g, b, 10, 1000*time.Millisecond)

	case agents.EventContextCompacting:
		// Flash — compacting context.
		c.Flash(packed, 500*time.Millisecond)

	case agents.EventNotification:
		c.Flash(packed, 200*time.Millisecond)

	case agents.EventUnknown:
		// Silently ignore unrecognized events.
	}
}

func main() {
	// Load user config (falls back to defaults if no file).
	cfg := config.Load()

	// Read the hook JSON from stdin.
	var raw json.RawMessage
	if err := json.NewDecoder(os.Stdin).Decode(&raw); err != nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] Failed to read stdin: %v\n", err)
		os.Exit(0) // Exit 0 so we don't block the agent.
	}

	// Auto-detect which agent sent this event.
	adapter := agents.DetectAdapter(raw)
	if adapter == nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] No adapter matched the input JSON\n")
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "[agent-chroma] Detected agent: %s\n", adapter.Name())

	// Parse the raw JSON into a canonical event.
	event, err := adapter.ParseEvent(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] Failed to parse event: %v\n", err)
		os.Exit(0)
	}

	// Initialize Chroma SDK session.
	client, err := chroma.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] Chroma SDK init failed: %v\n", err)
		os.Exit(0)
	}
	defer client.Close()

	// Route the canonical event to the appropriate effect.
	handleEvent(client, event, cfg)

	// Always exit 0 — we never want to block the agent's workflow.
	os.Exit(0)
}
