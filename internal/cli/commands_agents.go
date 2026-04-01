package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/settixx/claude-code-go/internal/tui"
)

func cmdAgents(_ string, ctx *CommandContext) error {
	if ctx.StateStore == nil {
		fmt.Fprintln(os.Stdout, tui.Dim("(Not yet connected to an agent registry)"))
		return nil
	}

	appState := ctx.StateStore.Get()
	if len(appState.AgentNameRegistry) == 0 {
		fmt.Fprintln(os.Stdout, tui.Dim("No agents registered."))
		return nil
	}

	names := make([]string, 0, len(appState.AgentNameRegistry))
	for name := range appState.AgentNameRegistry {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintln(os.Stdout, tui.Bold("Active agents:"))
	fmt.Fprintln(os.Stdout)
	for _, name := range names {
		agentID := appState.AgentNameRegistry[name]
		fmt.Fprintf(os.Stdout, "  %-20s %s\n", tui.Cyan(name), tui.Dim(string(agentID)))
	}
	fmt.Fprintln(os.Stdout)
	return nil
}
