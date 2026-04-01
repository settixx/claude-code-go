package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
)

func registerSessionCommands(reg *CommandRegistry) {
	reg.Register(&Command{
		Name:        "resume",
		Description: "List and resume previous sessions",
		Handler:     cmdResume,
	})
	reg.Register(&Command{
		Name:        "session",
		Description: "Session management (list, info)",
		Handler:     cmdSession,
	})
	reg.Register(&Command{
		Name:        "export",
		Description: "Export conversation to file",
		Handler:     cmdExport,
	})
	reg.Register(&Command{
		Name:        "memory",
		Description: "Show or edit memory files",
		Handler:     cmdMemory,
	})
}

func cmdResume(args string, _ *CommandContext) error {
	if args != "" {
		fmt.Fprintf(os.Stdout, "%s Resuming session %s… (not yet implemented)\n", tui.Blue("ℹ"), tui.Cyan(args))
		return nil
	}
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Session resume not yet implemented. Usage: /resume <session-id>"))
	return nil
}

func cmdSession(args string, ctx *CommandContext) error {
	switch args {
	case "", "info":
		sid := ctx.SessionID
		if sid == "" {
			sid = "(none)"
		}
		fmt.Fprintln(os.Stdout, tui.Bold("Current session:"))
		fmt.Fprintf(os.Stdout, "  id: %s\n", sid)
	case "list":
		fmt.Fprintln(os.Stdout, tui.Blue("ℹ Session listing not yet implemented."))
	default:
		fmt.Fprintf(os.Stdout, "Usage: /session [info|list]\n")
	}
	return nil
}

func cmdExport(args string, _ *CommandContext) error {
	target := args
	if target == "" {
		target = "conversation.md"
	}
	fmt.Fprintf(os.Stdout, "%s Export to %s not yet implemented.\n", tui.Blue("ℹ"), tui.Cyan(target))
	return nil
}

func cmdMemory(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Memory file management not yet implemented."))
	return nil
}
