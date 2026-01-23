package agent

// ToolCallEvent represents a tool lifecycle event from ACP.
// Used to track tool calls from pending → in_progress → completed.
type ToolCallEvent struct {
	ToolCallID string         // Stable ID for tracking updates
	Title      string         // Tool name (e.g., "bash")
	Status     string         // "pending", "in_progress", "completed"
	RawInput   map[string]any // Command params (populated on in_progress+)
	Output     string         // Tool output (populated on completed)
	Kind       string         // "execute", etc.
}
