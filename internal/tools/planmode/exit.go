package planmode

import (
	"context"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// ExitTool switches the session out of plan mode back to default.
type ExitTool struct {
	toolutil.BaseTool
	state interfaces.StateStore
}

func NewExitTool(state interfaces.StateStore) *ExitTool {
	return &ExitTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "ExitPlanMode",
			ToolAliases:     []string{"exit_plan_mode"},
			ToolSearchHint:  "plan mode exit leave",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		state: state,
	}
}

func (t *ExitTool) Description(_ map[string]interface{}) (string, error) {
	return "Exit plan mode and return to normal operation. Write operations will be allowed again.", nil
}

func (t *ExitTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
	}
}

func (t *ExitTool) Call(_ context.Context, _ map[string]interface{}) (*types.ToolResult, error) {
	t.state.Update(func(s *types.AppState) {
		s.PermissionMode = types.PermDefault
	})
	return &types.ToolResult{Data: "Exited plan mode. Write operations are now allowed."}, nil
}
