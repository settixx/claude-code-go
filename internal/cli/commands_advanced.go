package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
)

func registerAdvancedCommands(reg *CommandRegistry) {
	reg.Register(&Command{
		Name:        "permissions",
		Description: "Show or change permission mode",
		Handler:     cmdPermissions,
	})
	reg.Register(&Command{
		Name:        "mcp",
		Description: "List connected MCP servers and their status",
		Handler:     cmdMCP,
	})
	reg.Register(&Command{
		Name:        "plan",
		Description: "Toggle plan mode (disables write tools)",
		Handler:     cmdPlan,
	})
	reg.Register(&Command{
		Name:        "tasks",
		Description: "List running and completed tasks",
		Handler:     cmdTasks,
	})
	reg.Register(&Command{
		Name:        "agents",
		Description: "List active agents",
		Handler:     cmdAgents,
	})
	reg.Register(&Command{
		Name:        "skills",
		Description: "List available skills",
		Handler:     cmdSkills,
	})
	reg.Register(&Command{
		Name:        "plugins",
		Description: "Plugin management (list/enable/disable)",
		Handler:     cmdPlugins,
	})
	reg.Register(&Command{
		Name:        "theme",
		Description: "Select display theme",
		Handler:     cmdTheme,
	})
	reg.Register(&Command{
		Name:        "vim",
		Description: "Toggle vim keybinding mode",
		Handler:     cmdVim,
	})
	reg.Register(&Command{
		Name:        "buddy",
		Description: "Toggle companion buddy display",
		Handler:     cmdBuddy,
	})
	reg.Register(&Command{
		Name:        "review",
		Description: "Show git diff for code review",
		Handler:     cmdReview,
	})
}

func cmdPermissions(args string, ctx *CommandContext) error {
	if args == "" {
		fmt.Fprintf(os.Stdout, "Current permission mode: %s\n", tui.Cyan(ctx.PermissionMode))
		fmt.Fprintln(os.Stdout, tui.Dim("  Available: default, plan, acceptEdits, bypassPermissions, auto"))
		return nil
	}
	ctx.PermissionMode = args
	fmt.Fprintf(os.Stdout, "Permission mode changed to: %s\n", tui.Cyan(args))
	return nil
}

func cmdPlugins(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Plugin management not yet implemented."))
	return nil
}

func cmdTheme(args string, _ *CommandContext) error {
	if args == "" {
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ Theme selection not yet implemented. Usage: /theme <name>"))
		return nil
	}
	fmt.Fprintf(os.Stdout, "%s Theme switch to %s not yet implemented.\n", tui.Blue("ℹ"), tui.Cyan(args))
	return nil
}

func cmdVim(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Vim mode toggle not yet implemented."))
	return nil
}
