package interfaces

import (
	"context"

	"github.com/settixx/claude-code-go/internal/types"
)

// LLMClient abstracts communication with an LLM provider.
type LLMClient interface {
	// Stream sends a query and returns a channel of streaming events.
	Stream(ctx context.Context, config types.QueryConfig, messages []types.Message) (<-chan types.StreamEvent, error)

	// Send performs a non-streaming request and returns the complete response.
	Send(ctx context.Context, config types.QueryConfig, messages []types.Message) (*types.APIMessage, error)

	// CountTokens estimates the token count for a set of messages.
	CountTokens(ctx context.Context, config types.QueryConfig, messages []types.Message) (int, error)
}

// ToolExecutor manages and runs tools.
type ToolExecutor interface {
	// Register adds a tool to the executor.
	Register(tool types.Tool)

	// Execute runs a named tool with the given input.
	Execute(ctx context.Context, name string, input map[string]interface{}) (*types.ToolResult, error)

	// Find looks up a tool by name or alias.
	Find(name string) types.Tool

	// All returns every registered tool.
	All() []types.Tool
}

// SessionStorage persists conversation histories.
type SessionStorage interface {
	// Save writes a session's messages to disk.
	Save(sessionID types.SessionId, messages []types.Message) error

	// Load reads a session's messages from disk.
	Load(sessionID types.SessionId) ([]types.Message, error)

	// List returns available session IDs, most recent first.
	List() ([]SessionInfo, error)

	// Delete removes a stored session.
	Delete(sessionID types.SessionId) error
}

// SessionInfo is a summary entry for listing sessions.
type SessionInfo struct {
	ID        types.SessionId `json:"id"`
	Title     string          `json:"title"`
	UpdatedAt int64           `json:"updated_at"`
	MessageCount int          `json:"message_count"`
}

// StateStore provides atomic reads and writes to AppState.
type StateStore interface {
	// Get returns a snapshot of the current state.
	Get() types.AppState

	// Update applies a mutation function atomically.
	Update(fn func(*types.AppState))

	// Subscribe registers a callback for state changes.
	Subscribe(fn func(types.AppState)) (unsubscribe func())
}

// ConfigProvider reads and watches configuration.
type ConfigProvider interface {
	// GetSettings returns the fully resolved settings.
	GetSettings() types.Settings

	// GetProjectConfig returns the project-level config.
	GetProjectConfig() types.ProjectConfig

	// GetUserConfig returns the user-level config.
	GetUserConfig() types.UserConfig

	// Reload re-reads all config sources.
	Reload() error
}

// Renderer abstracts terminal output.
type Renderer interface {
	// RenderMessage outputs a conversation message.
	RenderMessage(msg types.Message)

	// RenderError outputs an error.
	RenderError(err error)

	// RenderSpinner starts or updates a spinner.
	RenderSpinner(text string)

	// StopSpinner removes the spinner.
	StopSpinner()
}

// PermissionChecker evaluates tool invocations against the permission policy.
type PermissionChecker interface {
	// Check returns whether a tool invocation is allowed.
	Check(toolName string, input map[string]interface{}) types.PermissionResult

	// Mode returns the current permission mode.
	Mode() types.PermissionMode

	// SetMode changes the active permission mode.
	SetMode(mode types.PermissionMode)
}
