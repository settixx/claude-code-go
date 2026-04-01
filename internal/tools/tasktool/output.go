package tasktool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// OutputTool retrieves the output/log of a specific task.
type OutputTool struct {
	toolutil.BaseTool
	store *TaskStore
}

func NewOutputTool(store *TaskStore) *OutputTool {
	return &OutputTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "TaskOutput",
			ToolAliases:     []string{"task_output", "task_log"},
			ToolSearchHint:  "task output log result",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		store: store,
	}
}

func (t *OutputTool) Description(_ map[string]interface{}) (string, error) {
	return "Retrieve the output or log of a specific task.", nil
}

func (t *OutputTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the task whose output to retrieve.",
			},
		},
		Required: []string{"task_id"},
	}
}

func (t *OutputTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	id, err := toolutil.RequireString(input, "task_id")
	if err != nil {
		return nil, err
	}

	entry, ok := t.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}

	output := entry.Output
	if output == "" {
		output = "(no output yet)"
	}
	return &types.ToolResult{Data: output}, nil
}
