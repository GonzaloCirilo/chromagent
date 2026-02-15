package cursor

import (
	"encoding/json"
	"fmt"

	"github.com/GonzaloCirilo/agent-chroma/internal/agents"
)

func init() {
	agents.Register(&Adapter{})
}

// Adapter handles Cursor agent events.
type Adapter struct{}

func (a *Adapter) Name() string { return "cursor" }

// cursorPayload represents the JSON shape Cursor emits.
type cursorPayload struct {
	Event          *string `json:"event"`
	ConversationID *string `json:"conversation_id"`
}

// Detect returns true if the JSON contains cursor-specific fields.
func (a *Adapter) Detect(raw json.RawMessage) bool {
	var probe cursorPayload
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	// Cursor payloads have "event" + "conversation_id".
	return probe.Event != nil && probe.ConversationID != nil
}

// ParseEvent maps a Cursor event to a canonical AgentEvent.
// Mapping based on peon-ping's cursor adapter.
func (a *Adapter) ParseEvent(raw json.RawMessage) (agents.AgentEvent, error) {
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return agents.EventUnknown, fmt.Errorf("parse cursor event: %w", err)
	}

	if payload.Event == nil {
		return agents.EventUnknown, nil
	}

	switch *payload.Event {
	case "stop", "afterFileEdit":
		return agents.EventTaskComplete, nil
	case "beforeShellExecution", "beforeMCPExecution":
		return agents.EventToolExecuting, nil
	case "afterShellExecution", "afterMCPExecution":
		return agents.EventToolCompleted, nil
	case "start":
		return agents.EventSessionStart, nil
	case "end":
		return agents.EventSessionEnd, nil
	case "beforeReadFile":
		return agents.EventUnknown, nil
	default:
		return agents.EventUnknown, nil
	}
}
