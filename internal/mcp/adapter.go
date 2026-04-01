package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// ManagerAdapter wraps a Manager so it satisfies the tool-level interfaces
// (MCPCaller, ResourceLister, ResourceReader) without modifying Manager itself.
type ManagerAdapter struct {
	mgr *Manager
}

// NewManagerAdapter returns an adapter that delegates to mgr.
func NewManagerAdapter(mgr *Manager) *ManagerAdapter {
	return &ManagerAdapter{mgr: mgr}
}

// CallTool locates the named server's client and invokes the specified tool,
// returning the concatenated text content of the result.
func (a *ManagerAdapter) CallTool(ctx context.Context, server, tool string, args map[string]interface{}) (string, error) {
	client, err := a.client(server)
	if err != nil {
		return "", err
	}

	result, err := client.CallTool(ctx, tool, args)
	if err != nil {
		return "", fmt.Errorf("call tool %s on %s: %w", tool, server, err)
	}

	return formatToolResult(result), nil
}

// ListResources returns the resources exposed by a single named server.
func (a *ManagerAdapter) ListResources(ctx context.Context, server string) ([]types.ServerResource, error) {
	client, err := a.client(server)
	if err != nil {
		return nil, err
	}

	raw, err := client.ListResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("list resources from %s: %w", server, err)
	}

	out := make([]types.ServerResource, len(raw))
	for i, r := range raw {
		out[i] = types.ServerResource{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MimeType:    r.MimeType,
		}
	}
	return out, nil
}

// ConnectedServers returns the names of all servers currently registered.
func (a *ManagerAdapter) ConnectedServers() []string {
	a.mgr.mu.RLock()
	defer a.mgr.mu.RUnlock()

	names := make([]string, 0, len(a.mgr.clients))
	for name := range a.mgr.clients {
		names = append(names, name)
	}
	return names
}

// ReadResource fetches a single resource by URI from the named server.
func (a *ManagerAdapter) ReadResource(ctx context.Context, server, uri string) (string, error) {
	client, err := a.client(server)
	if err != nil {
		return "", err
	}
	return client.ReadResource(ctx, uri)
}

// client resolves a Client by server name, returning a clear error if absent.
func (a *ManagerAdapter) client(server string) (*Client, error) {
	a.mgr.mu.RLock()
	c, ok := a.mgr.clients[server]
	a.mgr.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("MCP server %q not found", server)
	}
	return c, nil
}

// formatToolResult concatenates all text content blocks from a ToolCallResult.
func formatToolResult(r *ToolCallResult) string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	var parts []string
	for _, c := range r.Content {
		if c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}
