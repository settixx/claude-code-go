package tasktool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// CreateTool creates a new task entry in the store.
type CreateTool struct {
	toolutil.BaseTool
	store *TaskStore
}

func NewCreateTool(store *TaskStore) *CreateTool {
	return &CreateTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "TaskCreate",
			ToolAliases:     []string{"task_create"},
			ToolSearchHint:  "task create new add",
			ConcurrencySafe: true,
		},
		store: store,
	}
}

func (t *CreateTool) Description(_ map[string]interface{}) (string, error) {
	return "Create a new tracked task entry. Returns the task ID.", nil
}

func (t *CreateTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable description of the task.",
			},
			"agent_type": map[string]interface{}{
				"type":        "string",
				"description": "Optional agent type for this task.",
			},
		},
		Required: []string{"description"},
	}
}

func (t *CreateTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	desc, err := toolutil.RequireString(input, "description")
	if err != nil {
		return nil, err
	}
	agentType := toolutil.OptionalString(input, "agent_type", "")

	entry := t.store.Create(desc, agentType)
	return &types.ToolResult{
		Data: fmt.Sprintf("Task created.\n  ID: %s\n  Description: %s\n  Status: %s", entry.ID, entry.Description, entry.Status),
	}, nil
}
