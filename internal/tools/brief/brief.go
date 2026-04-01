package brief

import (
	"context"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "Brief"

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"brief_output"},
			ToolSearchHint:  "brief mode, short output, compact response",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Return content in brief mode. " +
		"Simply passes through the provided content for concise output.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to output in brief mode.",
			},
		},
		Required: []string{"content"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	content, err := toolutil.RequireString(input, "content")
	if err != nil {
		return nil, err
	}

	return &types.ToolResult{Data: content}, nil
}
