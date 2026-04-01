package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// MCPPrompt describes a prompt template exposed by an MCP server.
type MCPPrompt struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Arguments   []MCPPromptArg `json:"arguments,omitempty"`
}

// MCPPromptArg describes one argument of an MCP prompt.
type MCPPromptArg struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// MCPPromptMessage is a single message in a prompt result.
type MCPPromptMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// PromptResult is the server's response to "prompts/get".
type PromptResult struct {
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages"`
}

// PromptsListResult is the server's response to "prompts/list".
type PromptsListResult struct {
	Prompts []MCPPrompt `json:"prompts"`
}

// PromptsGetParams are sent with the "prompts/get" request.
type PromptsGetParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// ListPrompts retrieves all prompts exposed by the server.
func (c *Client) ListPrompts(ctx context.Context) ([]MCPPrompt, error) {
	raw, err := c.roundTrip(ctx, MethodPromptsList, nil)
	if err != nil {
		return nil, err
	}
	return decodeResult[PromptsListResult](raw, func(r PromptsListResult) []MCPPrompt { return r.Prompts })
}

// GetPrompt invokes a prompt with the given arguments.
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*PromptResult, error) {
	params := PromptsGetParams{Name: name, Arguments: args}
	raw, err := c.roundTrip(ctx, MethodPromptsGet, params)
	if err != nil {
		return nil, err
	}
	var result PromptResult
	if err := remarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode prompt result: %w", err)
	}
	return &result, nil
}
