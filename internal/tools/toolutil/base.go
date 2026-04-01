package toolutil

import (
	"encoding/json"
	"fmt"

	"github.com/settixx/claude-code-go/internal/types"
)

// BaseTool provides default implementations for optional Tool interface methods.
// Embed it in concrete tools to avoid boilerplate.
type BaseTool struct {
	ToolName        string
	ToolAliases     []string
	ToolSearchHint  string
	ToolMaxChars    int
	ReadOnly        bool
	Destructive     bool
	ConcurrencySafe bool
}

// Name returns the tool's canonical identifier.
func (b *BaseTool) Name() string { return b.ToolName }

// Aliases returns alternative names for backward compatibility.
func (b *BaseTool) Aliases() []string {
	if b.ToolAliases == nil {
		return []string{}
	}
	return b.ToolAliases
}

// IsEnabled reports whether the tool is available; always true by default.
func (b *BaseTool) IsEnabled() bool { return true }

// IsReadOnly reports whether the invocation only reads state.
func (b *BaseTool) IsReadOnly(_ map[string]interface{}) bool { return b.ReadOnly }

// IsDestructive reports whether the invocation is irreversible.
func (b *BaseTool) IsDestructive(_ map[string]interface{}) bool { return b.Destructive }

// IsConcurrencySafe reports whether multiple invocations can overlap.
func (b *BaseTool) IsConcurrencySafe(_ map[string]interface{}) bool { return b.ConcurrencySafe }

// CheckPermissions allows the invocation by default.
func (b *BaseTool) CheckPermissions(_ map[string]interface{}, _ types.PermissionMode) types.PermissionResult {
	return types.PermissionResult{Allowed: true}
}

// MaxResultSizeChars returns the result ceiling; uses ToolMaxChars or 100000.
func (b *BaseTool) MaxResultSizeChars() int {
	if b.ToolMaxChars > 0 {
		return b.ToolMaxChars
	}
	return 100_000
}

// InterruptBehavior returns "cancel" by default.
func (b *BaseTool) InterruptBehavior() string { return "cancel" }

// SearchHint returns a keyword phrase for tool discovery.
func (b *BaseTool) SearchHint() string { return b.ToolSearchHint }

// FormatOutput serialises data to a human-friendly JSON string.
// Falls back to fmt.Sprintf for non-marshallable values.
func FormatOutput(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(b)
}

// TruncateResult clips s to maxChars, appending a notice when truncated.
func TruncateResult(s string, maxChars int) string {
	if maxChars <= 0 || len(s) <= maxChars {
		return s
	}
	const suffix = "\n... [result truncated]"
	cut := maxChars - len(suffix)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + suffix
}

// RequireString extracts a required string field from tool input.
func RequireString(input map[string]interface{}, key string) (string, error) {
	v, ok := input[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("field %q must be a string", key)
	}
	return s, nil
}

// OptionalString extracts an optional string field, returning fallback if absent.
func OptionalString(input map[string]interface{}, key, fallback string) string {
	v, ok := input[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	return s
}

// OptionalInt extracts an optional integer field from tool input.
// JSON numbers arrive as float64, so we accept both.
func OptionalInt(input map[string]interface{}, key string, fallback int) int {
	v, ok := input[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return fallback
	}
}

// OptionalBool extracts an optional boolean field from tool input.
func OptionalBool(input map[string]interface{}, key string, fallback bool) bool {
	v, ok := input[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}
