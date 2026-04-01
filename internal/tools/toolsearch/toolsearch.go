package toolsearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "ToolSearch"

type Tool struct {
	toolutil.BaseTool
	registry *types.ToolRegistry
}

func New(registry *types.ToolRegistry) *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"tool_search", "search_tools"},
			ToolSearchHint:  "find tools, discover available tools, tool lookup",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		registry: registry,
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Search through registered tools by name, description, or search hint. " +
		"Returns matching tool names and descriptions.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query to match against tool names, descriptions, and hints.",
			},
		},
		Required: []string{"query"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	query, err := toolutil.RequireString(input, "query")
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	matches := t.searchTools(queryLower)

	if len(matches) == 0 {
		return &types.ToolResult{Data: fmt.Sprintf("No tools found matching %q.", query)}, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d matching tool(s):\n\n", len(matches))
	for _, m := range matches {
		fmt.Fprintf(&b, "- **%s**: %s\n", m.name, m.desc)
	}
	return &types.ToolResult{Data: b.String()}, nil
}

type toolMatch struct {
	name string
	desc string
}

func (t *Tool) searchTools(queryLower string) []toolMatch {
	var matches []toolMatch
	for _, tool := range t.registry.All() {
		if matchesTool(tool, queryLower) {
			desc, _ := tool.Description(nil)
			matches = append(matches, toolMatch{name: tool.Name(), desc: desc})
		}
	}
	return matches
}

func matchesTool(tool types.Tool, queryLower string) bool {
	if containsLower(tool.Name(), queryLower) {
		return true
	}
	for _, alias := range tool.Aliases() {
		if containsLower(alias, queryLower) {
			return true
		}
	}
	if containsLower(tool.SearchHint(), queryLower) {
		return true
	}
	desc, _ := tool.Description(nil)
	return containsLower(desc, queryLower)
}

func containsLower(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), sub)
}
