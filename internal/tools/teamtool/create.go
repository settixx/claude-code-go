package teamtool

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// CreateTool creates a named team of coordinated agents.
type CreateTool struct {
	toolutil.BaseTool
	teams *coordinator.TeamManager
}

func NewCreateTool(teams *coordinator.TeamManager) *CreateTool {
	return &CreateTool{
		BaseTool: toolutil.BaseTool{
			ToolName:       "TeamCreate",
			ToolAliases:    []string{"team_create"},
			ToolSearchHint: "team create new group agents",
		},
		teams: teams,
	}
}

func (t *CreateTool) Description(_ map[string]interface{}) (string, error) {
	return "Create a named team of coordinated agents. Returns the team name.", nil
}

func (t *CreateTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name for the new team.",
			},
			"agents": map[string]interface{}{
				"type":        "array",
				"description": "Array of agent configs. Each item should have at least an 'id' field.",
				"items": map[string]interface{}{
					"type": "object",
				},
			},
		},
		Required: []string{"name", "agents"},
	}
}

func (t *CreateTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	name, err := toolutil.RequireString(input, "name")
	if err != nil {
		return nil, err
	}

	agentIDs, err := extractAgentIDs(input)
	if err != nil {
		return nil, err
	}
	if len(agentIDs) == 0 {
		return nil, fmt.Errorf("at least one agent is required")
	}

	team, err := t.teams.Create(name, agentIDs)
	if err != nil {
		return nil, err
	}

	return &types.ToolResult{
		Data: fmt.Sprintf("Team %q created with %d member(s). Leader: %s",
			team.Name, len(team.Members), team.Leader),
	}, nil
}

func extractAgentIDs(input map[string]interface{}) ([]types.AgentId, error) {
	raw, ok := input["agents"]
	if !ok {
		return nil, fmt.Errorf("missing required field \"agents\"")
	}

	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("\"agents\" must be an array")
	}

	ids := make([]types.AgentId, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("agents[%d] must be an object", i)
		}
		idVal, ok := m["id"]
		if !ok {
			return nil, fmt.Errorf("agents[%d] missing \"id\" field", i)
		}
		idStr, ok := idVal.(string)
		if !ok {
			return nil, fmt.Errorf("agents[%d].id must be a string", i)
		}
		ids = append(ids, types.AgentId(idStr))
	}
	return ids, nil
}
