package claude

import (
	"encoding/json"
	"fmt"

	"github.com/GonzaloCirilo/agent-chroma/internal/agents"
	"github.com/GonzaloCirilo/agent-chroma/internal/hooks"
)

func init() {
	agents.Register(&Adapter{})
}

// Adapter handles Claude Code hook events.
type Adapter struct{}

func (a *Adapter) Name() string { return "claude" }

// Detect returns true if the JSON contains "hook_event_name", which is
// present in every Claude Code hook payload.
func (a *Adapter) Detect(raw json.RawMessage) bool {
	var probe struct {
		HookEventName *string `json:"hook_event_name"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.HookEventName != nil
}

// ParseEvent maps a Claude Code hook event to a canonical AgentEvent.
func (a *Adapter) ParseEvent(raw json.RawMessage) (agents.AgentEvent, error) {
	var common hooks.CommonInput
	if err := json.Unmarshal(raw, &common); err != nil {
		return agents.EventUnknown, fmt.Errorf("parse claude event: %w", err)
	}

	switch common.HookEventName {
	case "SessionStart":
		return agents.EventSessionStart, nil
	case "SessionEnd":
		return agents.EventSessionEnd, nil
	case "UserPromptSubmit":
		return agents.EventPromptAcknowledged, nil
	case "PreToolUse":
		return agents.EventToolExecuting, nil
	case "PostToolUse":
		return agents.EventToolCompleted, nil
	case "Stop":
		return agents.EventTaskComplete, nil
	case "SubagentStop":
		return agents.EventSubtaskComplete, nil
	case "Notification":
		return a.parseNotification(raw)
	case "PreCompact":
		return agents.EventContextCompacting, nil
	default:
		return agents.EventUnknown, nil
	}
}

func (a *Adapter) parseNotification(raw json.RawMessage) (agents.AgentEvent, error) {
	var ev hooks.NotificationInput
	if err := json.Unmarshal(raw, &ev); err != nil {
		return agents.EventUnknown, fmt.Errorf("parse notification: %w", err)
	}

	switch ev.NotificationType {
	case "permission_prompt":
		return agents.EventNeedsAttention, nil
	case "idle_prompt":
		return agents.EventWaitingForInput, nil
	default:
		return agents.EventNotification, nil
	}
}
