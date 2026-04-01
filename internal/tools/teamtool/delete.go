package teamtool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// DeleteTool stops all agents in a team and removes the team.
type DeleteTool struct {
	toolutil.BaseTool
	teams *coordinator.TeamManager
}

func NewDeleteTool(teams *coordinator.TeamManager) *DeleteTool {
	return &DeleteTool{
		BaseTool: toolutil.BaseTool{
			ToolName:       "TeamDelete",
			ToolAliases:    []string{"team_delete", "team_remove"},
			ToolSearchHint: "team delete remove stop",
		},
		teams: teams,
	}
}

func (t *DeleteTool) Description(_ map[string]interface{}) (string, error) {
	return "Stop all agents in a team and remove the team.", nil
}

func (t *DeleteTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"team_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the team to delete.",
			},
		},
		Required: []string{"team_name"},
	}
}

func (t *DeleteTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	name, err := toolutil.RequireString(input, "team_name")
	if err != nil {
		return nil, err
	}

	if err := t.teams.ShutdownTeam(name); err != nil {
		return nil, err
	}
	return &types.ToolResult{
		Data: fmt.Sprintf("Team %q deleted. All member agents have been stopped.", name),
	}, nil
}
