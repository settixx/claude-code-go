package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
)

func registerDevCommands(reg *CommandRegistry) {
	reg.Register(&Command{
		Name:        "commit",
		Description: "Generate a commit message from staged changes",
		Handler:     cmdCommit,
	})
	reg.Register(&Command{
		Name:        "diff",
		Description: "Show git diff summary",
		Handler:     cmdDiff,
	})
	reg.Register(&Command{
		Name:        "review",
		Description: "Request a code review of recent changes",
		Handler:     cmdReview,
	})
	reg.Register(&Command{
		Name:        "doctor",
		Description: "Run environment diagnostics",
		Handler:     cmdDoctor,
	})
}

func cmdCommit(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Commit message generation not yet implemented."))
	return nil
}

func cmdDiff(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Git diff summary not yet implemented."))
	return nil
}

func cmdReview(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Code review not yet implemented."))
	return nil
}

func cmdDoctor(_ string, _ *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Bold("Doctor — environment check:"))
	checks := []struct{ name, result string }{
		{"Go version", "ok (runtime detected)"},
		{"Git", "not yet checked"},
		{"API key", "not yet checked"},
		{"Config files", "not yet checked"},
		{"MCP servers", "not yet checked"},
	}
	for _, c := range checks {
		fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim(c.name), c.result)
	}
	return nil
}
