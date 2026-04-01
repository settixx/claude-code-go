package websearch

import (
	"context"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "WebSearch"

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"web_search"},
			ToolSearchHint:  "search the web, find information online",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Search the web for real-time information. " +
		"Returns summarized results and relevant URLs. " +
		"Requires an API key to be configured.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query to look up on the web.",
			},
			"allowed_domains": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional list of domains to restrict search results to.",
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

	_ = query
	return &types.ToolResult{
		Data: "Web search not yet configured. " +
			"Set TICODE_WEBSEARCH_API_KEY to enable web search functionality.",
	}, nil
}
