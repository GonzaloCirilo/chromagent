package hooks

// CommonInput contains fields present in every hook event.
type CommonInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"` // "default" | "plan" | "acceptEdits" | "bypassPermissions"
	HookEventName  string `json:"hook_event_name"`
}

// SessionStartInput is sent when an agent starts or resumes a session.
// Matcher values: "startup", "resume", "clear", "compact"
type SessionStartInput struct {
	CommonInput
	Source    string `json:"source"`               // "startup" | "resume" | "clear" | "compact"
	Model     string `json:"model,omitempty"`      // model name if provided
	AgentType string `json:"agent_type,omitempty"` // optional agent type
}

// SessionEndInput is sent when a Claude Code session ends.
type SessionEndInput struct {
	CommonInput
	Reason string `json:"reason"` // "clear" | "logout" | "prompt_input_exit" | "other"
}

// UserPromptSubmitInput is sent when the user submits a prompt.
type UserPromptSubmitInput struct {
	CommonInput
	Prompt string `json:"prompt"`
}

// PreToolUseInput is sent before a tool executes.
type PreToolUseInput struct {
	CommonInput
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
	ToolUseID string                 `json:"tool_use_id"`
}

// PostToolUseInput is sent after a tool completes successfully.
type PostToolUseInput struct {
	CommonInput
	ToolName     string                 `json:"tool_name"`
	ToolInput    map[string]interface{} `json:"tool_input"`
	ToolResponse map[string]interface{} `json:"tool_response"`
	ToolUseID    string                 `json:"tool_use_id"`
}

// StopInput is sent when the main agent or a subagent finishes responding.
type StopInput struct {
	CommonInput
	StopHookActive bool `json:"stop_hook_active"`
}

// PreCompactInput is sent before a compact operation.
type PreCompactInput struct {
	CommonInput
	Trigger            string `json:"trigger"` // "manual" | "auto"
	CustomInstructions string `json:"custom_instructions"`
}

// NotificationInput is sent when Claude Code fires a notification.
type NotificationInput struct {
	CommonInput
	Message          string `json:"message"`
	NotificationType string `json:"notification_type"` // "permission_prompt" | "idle_prompt" | "auth_success" | "elicitation_dialog"
}
