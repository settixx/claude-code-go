package listmcpresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// ResourceLister abstracts listing MCP resources across connected servers.
type ResourceLister interface {
	ListResources(ctx context.Context, server string) ([]types.ServerResource, error)
	ConnectedServers() []string
}

// Tool lists available resources from connected MCP servers.
type Tool struct {
	toolutil.BaseTool
	lister ResourceLister
}

func New(lister ResourceLister) *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "ListMCPResources",
			ToolAliases:     []string{"list_mcp_resources"},
			ToolSearchHint:  "mcp resource list server",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
		lister: lister,
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "List available resources from connected MCP servers. " +
		"Optionally filter by server name.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"server": map[string]interface{}{
				"type":        "string",
				"description": "Optional server name to filter resources by.",
			},
		},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	filter := toolutil.OptionalString(input, "server", "")

	servers := t.lister.ConnectedServers()
	if filter != "" {
		servers = []string{filter}
	}

	var buf strings.Builder
	for _, srv := range servers {
		resources, err := t.lister.ListResources(ctx, srv)
		if err != nil {
			fmt.Fprintf(&buf, "## %s\nError: %v\n\n", srv, err)
			continue
		}
		if len(resources) == 0 {
			fmt.Fprintf(&buf, "## %s\n(no resources)\n\n", srv)
			continue
		}
		fmt.Fprintf(&buf, "## %s\n", srv)
		for _, r := range resources {
			fmt.Fprintf(&buf, "- **%s** `%s`", r.Name, r.URI)
			if r.Description != "" {
				fmt.Fprintf(&buf, " — %s", r.Description)
			}
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}

	if buf.Len() == 0 {
		return &types.ToolResult{Data: "No MCP servers connected."}, nil
	}
	return &types.ToolResult{Data: buf.String()}, nil
}
