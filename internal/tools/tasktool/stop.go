package tasktool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// StopTool cancels a running or pending task.
type StopTool struct {
	toolutil.BaseTool
	store *TaskStore
}

func NewStopTool(store *TaskStore) *StopTool {
	return &StopTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "TaskStop",
			ToolAliases:     []string{"task_stop", "task_cancel"},
			ToolSearchHint:  "task stop cancel abort",
			ConcurrencySafe: true,
		},
		store: store,
	}
}

func (t *StopTool) Description(_ map[string]interface{}) (string, error) {
	return "Stop (cancel) a running or pending task.", nil
}

func (t *StopTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the task to stop.",
			},
		},
		Required: []string{"task_id"},
	}
}

func (t *StopTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	id, err := toolutil.RequireString(input, "task_id")
	if err != nil {
		return nil, err
	}

	if err := t.store.Stop(id); err != nil {
		return nil, err
	}
	return &types.ToolResult{
		Data: fmt.Sprintf("Task %s stopped.", id),
	}, nil
}
