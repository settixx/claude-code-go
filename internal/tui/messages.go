package tui

// DisplayMessage is a rendered conversation entry shown in the viewport.
type DisplayMessage struct {
	Role     string  // "user", "assistant", "system", "tool"
	Content  string
	ToolName string
	Tokens   int
	Cost     float64
}

// --- Custom Bubble Tea messages for async events ---

// StreamChunkMsg carries a partial text chunk from the streaming API.
type StreamChunkMsg struct{ Text string }

// StreamDoneMsg signals that the current stream has finished.
type StreamDoneMsg struct{}

// ToolCallMsg notifies the TUI that a tool invocation has started.
type ToolCallMsg struct {
	Name  string
	Input string
}

// ToolResultMsg delivers the output of a completed tool call.
type ToolResultMsg struct {
	Name   string
	Result string
}

// PermissionRequestMsg asks the user to approve or deny a tool invocation.
// The sender blocks on ResponseCh until the dialog resolves.
type PermissionRequestMsg struct {
	Tool       string
	Input      string
	ResponseCh chan bool
}

// ErrorMsg wraps an error that occurred during an async operation.
type ErrorMsg struct{ Err error }

// TokenUsageMsg carries token usage data from a completed message exchange.
type TokenUsageMsg struct {
	InputTokens  int
	OutputTokens int
}
