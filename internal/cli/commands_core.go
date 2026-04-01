package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/tui"
	"github.com/settixx/claude-code-go/internal/version"
)

// ErrExit is a sentinel indicating the user wants to exit the application.
var ErrExit = fmt.Errorf("exit")

func registerCoreCommands(reg *CommandRegistry) {
	reg.Register(&Command{
		Name:        "help",
		Description: "List all available commands",
		Handler:     cmdHelp,
	})
	reg.Register(&Command{
		Name:        "exit",
		Description: "Exit the application",
		Aliases:     []string{"quit"},
		Handler:     cmdExit,
	})
	reg.Register(&Command{
		Name:        "clear",
		Description: "Clear the terminal screen",
		Handler:     cmdClear,
	})
	reg.Register(&Command{
		Name:        "compact",
		Description: "Trigger conversation compaction",
		Handler:     cmdCompact,
	})
	reg.Register(&Command{
		Name:        "model",
		Description: "Show or change the current model",
		Handler:     cmdModel,
	})
	reg.Register(&Command{
		Name:        "config",
		Description: "Show or change configuration settings",
		Handler:     cmdConfig,
	})
	reg.Register(&Command{
		Name:        "version",
		Description: "Show version information",
		Handler:     cmdVersion,
	})
	reg.Register(&Command{
		Name:        "cost",
		Description: "Show token usage and estimated cost",
		Handler:     cmdCost,
	})
	reg.Register(&Command{
		Name:        "status",
		Description: "Show current session status",
		Handler:     cmdStatus,
	})
}

// defaultRegistry is lazily set so /help can enumerate all commands.
// RegisterDefaultCommands populates it during init.
var defaultRegistry *CommandRegistry

func setDefaultRegistry(reg *CommandRegistry) {
	defaultRegistry = reg
}

func cmdHelp(_ string, _ *CommandContext) error {
	reg := defaultRegistry
	if reg == nil {
		fmt.Fprintln(os.Stdout, "No commands registered.")
		return nil
	}

	cmds := reg.All()
	grouped := map[string][]*Command{
		"Core":     {},
		"Session":  {},
		"Dev":      {},
		"Advanced": {},
	}
	groupOrder := []string{"Core", "Session", "Dev", "Advanced"}

	coreNames := newSet("help", "exit", "clear", "compact", "model", "config", "version", "cost", "status")
	sessionNames := newSet("resume", "session", "export", "memory")
	devNames := newSet("commit", "diff", "review", "doctor")

	for _, cmd := range cmds {
		name := strings.TrimPrefix(cmd.Name, "/")
		switch {
		case coreNames[name]:
			grouped["Core"] = append(grouped["Core"], cmd)
		case sessionNames[name]:
			grouped["Session"] = append(grouped["Session"], cmd)
		case devNames[name]:
			grouped["Dev"] = append(grouped["Dev"], cmd)
		default:
			grouped["Advanced"] = append(grouped["Advanced"], cmd)
		}
	}

	fmt.Fprintln(os.Stdout, tui.Bold("Available commands:"))
	fmt.Fprintln(os.Stdout)
	for _, group := range groupOrder {
		list := grouped[group]
		if len(list) == 0 {
			continue
		}
		fmt.Fprintln(os.Stdout, tui.BoldUnderline(group))
		for _, cmd := range list {
			aliasStr := ""
			if len(cmd.Aliases) > 0 {
				aliasStr = tui.Dim(" (/" + strings.Join(cmd.Aliases, ", /") + ")")
			}
			fmt.Fprintf(os.Stdout, "  %-14s %s%s\n", tui.Cyan("/"+cmd.Name), cmd.Description, aliasStr)
		}
		fmt.Fprintln(os.Stdout)
	}
	return nil
}

func cmdExit(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Dim("Goodbye!"))
	return ErrExit
}

func cmdClear(_ string, _ *CommandContext) error {
	fmt.Fprint(os.Stdout, "\033[2J\033[H")
	return nil
}

func cmdCompact(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Conversation compaction not yet implemented."))
	return nil
}

func cmdModel(args string, ctx *CommandContext) error {
	if args == "" {
		fmt.Fprintf(os.Stdout, "Current model: %s\n", tui.Cyan(ctx.Model))
		return nil
	}
	ctx.Model = args
	fmt.Fprintf(os.Stdout, "Model changed to: %s\n", tui.Cyan(args))
	return nil
}

func cmdConfig(args string, ctx *CommandContext) error {
	if args != "" {
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ Config modification not yet implemented."))
		return nil
	}
	fmt.Fprintln(os.Stdout, tui.Bold("Current configuration:"))
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("model"), ctx.Model)
	fmt.Fprintf(os.Stdout, "  %-18s %v\n", tui.Dim("verbose"), ctx.Verbose)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("permission_mode"), ctx.PermissionMode)
	return nil
}

func cmdVersion(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, version.Full())
	return nil
}

func cmdCost(_ string, ctx *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Bold("Token usage:"))
	fmt.Fprintf(os.Stdout, "  %-18s %d\n", tui.Dim("input tokens"), ctx.TokensIn)
	fmt.Fprintf(os.Stdout, "  %-18s %d\n", tui.Dim("output tokens"), ctx.TokensOut)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("estimated cost"), tui.FormatCost(ctx.CostUSD))
	return nil
}

func cmdStatus(_ string, ctx *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Bold("Session status:"))
	sid := ctx.SessionID
	if sid == "" {
		sid = "(none)"
	}
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("session"), sid)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("model"), ctx.Model)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("permission_mode"), ctx.PermissionMode)
	fmt.Fprintf(os.Stdout, "  %-18s %v\n", tui.Dim("verbose"), ctx.Verbose)
	fmt.Fprintf(os.Stdout, "  %-18s %d in / %d out\n", tui.Dim("tokens"), ctx.TokensIn, ctx.TokensOut)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("cost"), tui.FormatCost(ctx.CostUSD))
	return nil
}

func newSet(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}
