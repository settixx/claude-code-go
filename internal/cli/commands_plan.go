package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
	"github.com/settixx/claude-code-go/internal/types"
)

func cmdPlan(_ string, ctx *CommandContext) error {
	if ctx.PermissionMode == "plan" {
		ctx.PermissionMode = "default"
		if ctx.PermissionChecker != nil {
			ctx.PermissionChecker.SetMode(types.PermDefault)
		}
		fmt.Fprintf(os.Stdout, "%s Switched to %s mode — write tools re-enabled.\n",
			tui.Green("✓"), tui.Cyan("default"))
		return nil
	}

	ctx.PermissionMode = "plan"
	if ctx.PermissionChecker != nil {
		ctx.PermissionChecker.SetMode(types.PermPlan)
	}
	fmt.Fprintf(os.Stdout, "%s Switched to %s mode — write tools disabled.\n",
		tui.Yellow("⚠"), tui.Cyan("plan"))
	return nil
}
