package tasktool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

var validStatuses = map[string]bool{
	"pending":   true,
	"running":   true,
	"completed": true,
	"failed":    true,
	"stopped":   true,
}

// UpdateTool modifies the status and optional result of an existing task.
type UpdateTool struct {
	toolutil.BaseTool
	store *TaskStore
}

func NewUpdateTool(store *TaskStore) *UpdateTool {
	return &UpdateTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "TaskUpdate",
			ToolAliases:     []string{"task_update"},
			ToolSearchHint:  "task update status change",
			ConcurrencySafe: true,
		},
		store: store,
	}
}

func (t *UpdateTool) Description(_ map[string]interface{}) (string, error) {
	return "Update the status (and optionally result) of an existing task.", nil
}

func (t *UpdateTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the task to update.",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"pending", "running", "completed", "failed", "stopped"},
				"description": "New status for the task.",
			},
			"result": map[string]interface{}{
				"type":        "string",
				"description": "Optional result or summary text.",
			},
		},
		Required: []string{"task_id", "status"},
	}
}

func (t *UpdateTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	id, err := toolutil.RequireString(input, "task_id")
	if err != nil {
		return nil, err
	}
	status, err := toolutil.RequireString(input, "status")
	if err != nil {
		return nil, err
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status %q", status)
	}

	result := toolutil.OptionalString(input, "result", "")

	if err := t.store.Update(id, status, result); err != nil {
		return nil, err
	}
	return &types.ToolResult{
		Data: fmt.Sprintf("Task %s updated to status %q.", id, status),
	}, nil
}
