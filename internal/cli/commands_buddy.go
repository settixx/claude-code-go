package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
)

func cmdBuddy(_ string, ctx *CommandContext) error {
	ctx.BuddyEnabled = !ctx.BuddyEnabled

	if ctx.BuddyEnabled {
		fmt.Fprintf(os.Stdout, "%s Buddy display %s\n", tui.Green("✓"), tui.Cyan("enabled"))
		return nil
	}
	fmt.Fprintf(os.Stdout, "%s Buddy display %s\n", tui.Green("✓"), tui.Cyan("disabled"))
	return nil
}
