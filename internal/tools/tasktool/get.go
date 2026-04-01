package tasktool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// GetTool retrieves detailed information about a single task.
type GetTool struct {
	toolutil.BaseTool
	store *TaskStore
}

func NewGetTool(store *TaskStore) *GetTool {
	return &GetTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "TaskGet",
			ToolAliases:     []string{"task_get"},
			ToolSearchHint:  "task get detail info",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		store: store,
	}
}

func (t *GetTool) Description(_ map[string]interface{}) (string, error) {
	return "Get detailed information about a specific task by ID.", nil
}

func (t *GetTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the task to retrieve.",
			},
		},
		Required: []string{"task_id"},
	}
}

func (t *GetTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	id, err := toolutil.RequireString(input, "task_id")
	if err != nil {
		return nil, err
	}

	entry, ok := t.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}

	return &types.ToolResult{Data: toolutil.FormatOutput(entry)}, nil
}
