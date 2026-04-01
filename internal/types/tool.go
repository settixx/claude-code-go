package types

import "context"

// ToolInputSchema is the JSON Schema for a tool's input parameters.
type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ValidationResult is the outcome of validating tool input.
type ValidationResult struct {
	Valid     bool   `json:"valid"`
	Message   string `json:"message,omitempty"`
	ErrorCode int    `json:"error_code,omitempty"`
}

// PermissionResult describes whether a tool invocation is allowed.
type PermissionResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// ToolResult is the value returned after a tool executes.
type ToolResult struct {
	Data        interface{} `json:"data"`
	NewMessages []Message   `json:"new_messages,omitempty"`
}

// SearchReadInfo describes whether a tool invocation is a search/read op.
type SearchReadInfo struct {
	IsSearch bool
	IsRead   bool
	IsList   bool
}

// Tool is the interface every tool must implement.
type Tool interface {
	// Name returns the tool's canonical identifier.
	Name() string

	// Aliases returns alternative names for backward compatibility.
	Aliases() []string

	// Description returns a human-readable description for the model.
	Description(input map[string]interface{}) (string, error)

	// InputSchema returns the JSON Schema for this tool's input.
	InputSchema() ToolInputSchema

	// Call executes the tool with validated input and returns a result.
	Call(ctx context.Context, input map[string]interface{}) (*ToolResult, error)

	// IsEnabled reports whether the tool is available in the current env.
	IsEnabled() bool

	// IsReadOnly reports whether the invocation only reads state.
	IsReadOnly(input map[string]interface{}) bool

	// IsDestructive reports whether the invocation is irreversible.
	IsDestructive(input map[string]interface{}) bool

	// IsConcurrencySafe reports whether multiple invocations can overlap.
	IsConcurrencySafe(input map[string]interface{}) bool

	// CheckPermissions verifies the caller has permission for this input.
	CheckPermissions(input map[string]interface{}, mode PermissionMode) PermissionResult

	// MaxResultSizeChars is the ceiling before results are persisted to disk.
	MaxResultSizeChars() int

	// InterruptBehavior returns "cancel" or "block" for new-message handling.
	InterruptBehavior() string

	// SearchHint returns a short keyword phrase for tool discovery.
	SearchHint() string
}

// ToolRegistry holds the full set of tools available to the query engine.
type ToolRegistry struct {
	tools []Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{}
}

func (r *ToolRegistry) Register(t Tool) {
	r.tools = append(r.tools, t)
}

func (r *ToolRegistry) All() []Tool {
	return r.tools
}

func (r *ToolRegistry) Find(name string) Tool {
	for _, t := range r.tools {
		if t.Name() == name {
			return t
		}
		for _, a := range t.Aliases() {
			if a == name {
				return t
			}
		}
	}
	return nil
}
