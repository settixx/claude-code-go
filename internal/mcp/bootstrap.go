package mcp

import (
	"context"
	"log/slog"

	"github.com/settixx/claude-code-go/internal/types"
)

// BootstrapResult holds the results of MCP auto-startup.
type BootstrapResult struct {
	Manager      *Manager
	Tools        []types.Tool
	Instructions []ServerInstruction
}

// Bootstrap creates a Manager, registers all configured servers, connects
// them, and returns the aggregated tools and instructions.
//
// Individual server failures are logged as warnings but do not prevent
// other servers from connecting.
func Bootstrap(ctx context.Context, servers map[string]types.McpServerConfig) (*BootstrapResult, error) {
	mgr := NewManager()

	if len(servers) == 0 {
		return &BootstrapResult{Manager: mgr}, nil
	}

	for name, cfg := range servers {
		if err := mgr.AddServer(name, cfg); err != nil {
			slog.Warn("mcp: failed to add server", "name", name, "error", err)
		}
	}

	if err := mgr.ConnectAll(ctx); err != nil {
		slog.Warn("mcp: some servers failed to connect", "error", err)
	}

	return &BootstrapResult{
		Manager:      mgr,
		Tools:        mgr.AllTools(),
		Instructions: mgr.AllInstructions(),
	}, nil
}
