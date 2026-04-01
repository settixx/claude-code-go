package planmode

import (
	"context"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// EnterTool switches the session into plan / read-only mode.
type EnterTool struct {
	toolutil.BaseTool
	state interfaces.StateStore
}

func NewEnterTool(state interfaces.StateStore) *EnterTool {
	return &EnterTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "EnterPlanMode",
			ToolAliases:     []string{"enter_plan_mode", "plan_mode"},
			ToolSearchHint:  "plan mode read only enter",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		state: state,
	}
}

func (t *EnterTool) Description(_ map[string]interface{}) (string, error) {
	return "Enter plan mode (read-only). No write operations are allowed until you exit plan mode.", nil
}

func (t *EnterTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
	}
}

func (t *EnterTool) Call(_ context.Context, _ map[string]interface{}) (*types.ToolResult, error) {
	t.state.Update(func(s *types.AppState) {
		s.PermissionMode = types.PermPlan
	})
	return &types.ToolResult{Data: "Entered plan mode. All write operations are now blocked."}, nil
}
