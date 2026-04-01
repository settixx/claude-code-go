package cli

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/permissions"
	"github.com/settixx/claude-code-go/internal/types"
)

// PermissionAwareExecutor wraps a ToolRegistry with a permission Checker.
// It implements interfaces.ToolExecutor, gating every Execute call through
// the permission system before delegating to the underlying tool.
type PermissionAwareExecutor struct {
	registry *types.ToolRegistry
	checker  *permissions.Checker
}

// NewPermissionAwareExecutor creates an executor that checks permissions
// before invoking tools.
func NewPermissionAwareExecutor(registry *types.ToolRegistry, checker *permissions.Checker) *PermissionAwareExecutor {
	return &PermissionAwareExecutor{registry: registry, checker: checker}
}

func (e *PermissionAwareExecutor) Register(t types.Tool) {
	e.registry.Register(t)
}

func (e *PermissionAwareExecutor) All() []types.Tool {
	return e.registry.All()
}

func (e *PermissionAwareExecutor) Find(name string) types.Tool {
	return e.registry.Find(name)
}

func (e *PermissionAwareExecutor) Execute(ctx context.Context, name string, input map[string]interface{}) (*types.ToolResult, error) {
	tool := e.registry.Find(name)
	if tool == nil {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	result, err := e.checker.CheckWithPrompt(ctx, name, input)
	if err != nil {
		return nil, fmt.Errorf("permission check failed for %s: %w", name, err)
	}
	if !result.Allowed {
		return nil, &PermissionDeniedError{Tool: name, Reason: result.Reason}
	}

	return tool.Call(ctx, input)
}

// PermissionDeniedError is returned when a tool invocation is blocked by the
// permission system.
type PermissionDeniedError struct {
	Tool   string
	Reason string
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied for tool %s: %s", e.Tool, e.Reason)
}
