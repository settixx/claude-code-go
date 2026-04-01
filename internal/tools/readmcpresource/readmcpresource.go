package readmcpresource

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// ResourceReader abstracts reading a single MCP resource.
type ResourceReader interface {
	ReadResource(ctx context.Context, server, uri string) (string, error)
}

// Tool reads a specific resource from an MCP server.
type Tool struct {
	toolutil.BaseTool
	reader ResourceReader
}

func New(reader ResourceReader) *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "ReadMCPResource",
			ToolAliases:     []string{"read_mcp_resource"},
			ToolSearchHint:  "mcp resource read fetch",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		reader: reader,
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Read a specific resource from a connected MCP server by URI.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"server": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to read from.",
			},
			"uri": map[string]interface{}{
				"type":        "string",
				"description": "The resource URI to read.",
			},
		},
		Required: []string{"server", "uri"},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	server, err := toolutil.RequireString(input, "server")
	if err != nil {
		return nil, err
	}
	uri, err := toolutil.RequireString(input, "uri")
	if err != nil {
		return nil, err
	}

	content, err := t.reader.ReadResource(ctx, server, uri)
	if err != nil {
		return nil, fmt.Errorf("read resource %s from %s: %w", uri, server, err)
	}
	return &types.ToolResult{Data: content}, nil
}
