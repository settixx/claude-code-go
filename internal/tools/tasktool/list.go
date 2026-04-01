package tasktool

import (
	"context"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// ListTool returns a formatted list of all tracked tasks.
type ListTool struct {
	toolutil.BaseTool
	store *TaskStore
}

func NewListTool(store *TaskStore) *ListTool {
	return &ListTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "TaskList",
			ToolAliases:     []string{"task_list"},
			ToolSearchHint:  "task list all show",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		store: store,
	}
}

func (t *ListTool) Description(_ map[string]interface{}) (string, error) {
	return "List tracked tasks, optionally filtered by status.", nil
}

func (t *ListTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"pending", "running", "completed", "failed", "stopped"},
				"description": "Optional status filter.",
			},
		},
	}
}

func (t *ListTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	status := toolutil.OptionalString(input, "status", "")
	tasks := t.store.List(status)

	if len(tasks) == 0 {
		msg := "No tasks found."
		if status != "" {
			msg = fmt.Sprintf("No tasks with status %q.", status)
		}
		return &types.ToolResult{Data: msg}, nil
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "Tasks (%d):\n", len(tasks))
	for _, e := range tasks {
		fmt.Fprintf(&buf, "  [%s] %s — %s\n", e.Status, e.ID, e.Description)
	}
	return &types.ToolResult{Data: buf.String()}, nil
}
