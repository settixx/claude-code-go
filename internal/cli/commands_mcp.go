package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
)

func cmdMCP(_ string, ctx *CommandContext) error {
	if ctx.StateStore == nil {
		fmt.Fprintln(os.Stdout, tui.Dim("(Not yet connected to MCP state)"))
		return nil
	}

	appState := ctx.StateStore.Get()
	clients := appState.MCP.Clients

	if len(clients) == 0 {
		fmt.Fprintln(os.Stdout, tui.Dim("No MCP servers connected."))
		return nil
	}

	fmt.Fprintln(os.Stdout, tui.Bold("MCP servers:"))
	fmt.Fprintln(os.Stdout)
	for _, c := range clients {
		icon := mcpStatusIcon(c.Status)
		fmt.Fprintf(os.Stdout, "  %s %-24s %s\n", icon, tui.Cyan(c.Name), tui.Dim(c.Status))
	}

	toolCount := len(appState.MCP.Tools)
	resourceCount := 0
	for _, rs := range appState.MCP.Resources {
		resourceCount += len(rs)
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "  %s %d tools, %d resources\n",
		tui.Dim("Total:"), toolCount, resourceCount)
	fmt.Fprintln(os.Stdout)
	return nil
}

func mcpStatusIcon(status string) string {
	switch status {
	case "connected", "ready":
		return tui.Green("●")
	case "connecting", "initializing":
		return tui.Yellow("○")
	case "error", "failed", "disconnected":
		return tui.Red("✗")
	default:
		return tui.Dim("?")
	}
}
