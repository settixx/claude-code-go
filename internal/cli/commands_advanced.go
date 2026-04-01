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
		Description: "MCP server management (list/add/remove)",
		Handler:     cmdMCP,
	})
	reg.Register(&Command{
		Name:        "plan",
		Description: "Enter plan mode for structured reasoning",
		Handler:     cmdPlan,
	})
	reg.Register(&Command{
		Name:        "tasks",
		Description: "Task management (list/create/cancel)",
		Handler:     cmdTasks,
	})
	reg.Register(&Command{
		Name:        "agents",
		Description: "Agent management (list/select)",
		Handler:     cmdAgents,
	})
	reg.Register(&Command{
		Name:        "skills",
		Description: "Skill management (list/info)",
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
		Description: "Companion management",
		Handler:     cmdBuddy,
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

func cmdMCP(args string, _ *CommandContext) error {
	switch args {
	case "", "list":
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ No MCP servers connected. (management not yet implemented)"))
	case "add":
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ MCP server add not yet implemented."))
	case "remove":
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ MCP server remove not yet implemented."))
	default:
		fmt.Fprintln(os.Stdout, "Usage: /mcp [list|add|remove]")
	}
	return nil
}

func cmdPlan(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Plan mode not yet implemented."))
	return nil
}

func cmdTasks(args string, _ *CommandContext) error {
	switch args {
	case "", "list":
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ No active tasks. (task management not yet implemented)"))
	default:
		fmt.Fprintln(os.Stdout, "Usage: /tasks [list|create|cancel]")
	}
	return nil
}

func cmdAgents(args string, _ *CommandContext) error {
	switch args {
	case "", "list":
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ No agents registered. (agent management not yet implemented)"))
	default:
		fmt.Fprintln(os.Stdout, "Usage: /agents [list|select]")
	}
	return nil
}

func cmdSkills(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Skill management not yet implemented."))
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

func cmdBuddy(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Companion management not yet implemented."))
	return nil
}
