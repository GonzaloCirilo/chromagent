package agents

import "encoding/json"

// AgentEvent represents a canonical agent state that maps to a Chroma effect.
type AgentEvent int

const (
	EventSessionStart      AgentEvent = iota // Agent booting up
	EventSessionEnd                          // Session over
	EventPromptAcknowledged                  // User input received
	EventToolExecuting                       // Tool/action in progress
	EventToolCompleted                       // Tool/action done
	EventTaskComplete                        // Agent finished task
	EventSubtaskComplete                     // Subagent/subtask done
	EventNeedsAttention                      // Permission/approval needed
	EventWaitingForInput                     // Idle, waiting for user
	EventContextCompacting                   // Compacting context
	EventNotification                        // Generic notification
	EventUnknown                             // Unrecognized event
)

// eventNames maps each AgentEvent to its snake_case config key.
var eventNames = map[AgentEvent]string{
	EventSessionStart:      "session_start",
	EventSessionEnd:        "session_end",
	EventPromptAcknowledged: "prompt_acknowledged",
	EventToolExecuting:     "tool_executing",
	EventToolCompleted:     "tool_completed",
	EventTaskComplete:      "task_complete",
	EventSubtaskComplete:   "subtask_complete",
	EventNeedsAttention:    "needs_attention",
	EventWaitingForInput:   "waiting_for_input",
	EventContextCompacting: "context_compacting",
	EventNotification:      "notification",
	EventUnknown:           "unknown",
}

// Name returns the snake_case config key for this event.
func (e AgentEvent) Name() string {
	if n, ok := eventNames[e]; ok {
		return n
	}
	return "unknown"
}

// AgentAdapter normalizes agent-specific JSON into canonical AgentEvents.
type AgentAdapter interface {
	// Name returns the adapter identifier (e.g., "claude", "cursor").
	Name() string
	// Detect returns true if the raw JSON looks like it came from this agent.
	Detect(raw json.RawMessage) bool
	// ParseEvent takes raw stdin JSON and returns a canonical AgentEvent.
	ParseEvent(raw json.RawMessage) (AgentEvent, error)
}

// registry holds all registered adapters in detection priority order.
var registry []AgentAdapter

// Register adds an adapter to the detection registry.
func Register(a AgentAdapter) {
	registry = append(registry, a)
}

// DetectAdapter tries each registered adapter's Detect method and returns the
// first match. Returns nil if no adapter recognizes the JSON.
func DetectAdapter(raw json.RawMessage) AgentAdapter {
	for _, a := range registry {
		if a.Detect(raw) {
			return a
		}
	}
	return nil
}
