package askuser

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "AskUserQuestion"

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"ask_user", "ask"},
			ToolSearchHint:  "ask user question, prompt, clarification",
			ReadOnly:        true,
			ConcurrencySafe: false,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Ask the user a question and wait for a response. " +
		"Use this when you need clarification or confirmation.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "The question to ask the user.",
			},
		},
		Required: []string{"question"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	question, err := toolutil.RequireString(input, "question")
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n[Question] %s\n", question)

	return &types.ToolResult{
		Data: "User response not available in non-interactive mode.",
	}, nil
}
