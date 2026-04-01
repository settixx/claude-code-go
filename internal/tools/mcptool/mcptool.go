package mcptool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// MCPCaller abstracts calling an MCP server tool so the real client
// can be injected without importing the full mcp package.
type MCPCaller interface {
	CallTool(ctx context.Context, server, tool string, args map[string]interface{}) (string, error)
}

// Tool exposes MCP server tools to the model.
type Tool struct {
	toolutil.BaseTool
	caller MCPCaller
}

func New(caller MCPCaller) *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:       "MCPTool",
			ToolAliases:    []string{"mcp_tool", "mcp"},
			ToolSearchHint: "mcp server tool call invoke",
		},
		caller: caller,
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Call a tool exposed by a connected MCP server.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"server_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to call.",
			},
			"tool_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the tool on the MCP server.",
			},
			"arguments": map[string]interface{}{
				"type":        "object",
				"description": "Arguments to pass to the MCP tool.",
			},
		},
		Required: []string{"server_name", "tool_name"},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	server, err := toolutil.RequireString(input, "server_name")
	if err != nil {
		return nil, err
	}
	tool, err := toolutil.RequireString(input, "tool_name")
	if err != nil {
		return nil, err
	}

	args := extractObject(input, "arguments")

	result, err := t.caller.CallTool(ctx, server, tool, args)
	if err != nil {
		return nil, fmt.Errorf("MCP call %s/%s: %w", server, tool, err)
	}
	return &types.ToolResult{Data: result}, nil
}

func extractObject(input map[string]interface{}, key string) map[string]interface{} {
	v, ok := input[key]
	if !ok {
		return nil
	}
	switch m := v.(type) {
	case map[string]interface{}:
		return m
	default:
		// JSON round-trip for non-map values (e.g. from some decoders)
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out map[string]interface{}
		if json.Unmarshal(b, &out) != nil {
			return nil
		}
		return out
	}
}
