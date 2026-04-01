package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// SkillTool exposes the skill registry as a types.Tool so the LLM can invoke skills.
type SkillTool struct {
	toolutil.BaseTool
	registry *SkillRegistry
}

// NewSkillTool creates a SkillTool wired to the given registry.
func NewSkillTool(registry *SkillRegistry) *SkillTool {
	return &SkillTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "skill",
			ToolAliases:     []string{"run_skill"},
			ToolSearchHint:  "skill invoke run",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		registry: registry,
	}
}

// Description returns a dynamic description that lists available skills.
func (t *SkillTool) Description(_ map[string]interface{}) (string, error) {
	invocable := t.registry.UserInvocable()
	if len(invocable) == 0 {
		return "Run a skill by name. No user-invocable skills are loaded.", nil
	}

	var b strings.Builder
	b.WriteString("Run a skill by name. Available skills:\n")
	for _, s := range invocable {
		fmt.Fprintf(&b, "  - %s: %s\n", s.Name, s.Description)
	}
	return b.String(), nil
}

// InputSchema defines the expected input: skill_name (required) and args (optional).
func (t *SkillTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"skill_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the skill to invoke",
			},
			"args": map[string]interface{}{
				"type":        "string",
				"description": "Optional arguments or context to pass to the skill",
			},
		},
		Required: []string{"skill_name"},
	}
}

// Call looks up the skill and returns its content for the LLM to follow.
func (t *SkillTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	name, err := toolutil.RequireString(input, "skill_name")
	if err != nil {
		return nil, err
	}
	args := toolutil.OptionalString(input, "args", "")

	s, ok := t.registry.Get(name)
	if !ok {
		return &types.ToolResult{
			Data: fmt.Sprintf("Unknown skill %q. Use the skill tool with a valid skill_name.", name),
		}, nil
	}

	var result strings.Builder
	fmt.Fprintf(&result, "# Skill: %s\n\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(&result, "%s\n\n", s.Description)
	}
	result.WriteString(s.Content)
	if args != "" {
		fmt.Fprintf(&result, "\n\n## User Context\n\n%s", args)
	}
	return &types.ToolResult{Data: result.String()}, nil
}
