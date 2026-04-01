package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/tui"
)

func cmdReview(_ string, ctx *CommandContext) error {
	cwd := effectiveCWD(ctx)

	staged, stagedErr := runGit(cwd, "diff", "--cached")
	unstaged, unstagedErr := runGit(cwd, "diff")

	if stagedErr != nil && unstagedErr != nil {
		return fmt.Errorf("failed to run git diff: %v", stagedErr)
	}

	hasStagedDiff := strings.TrimSpace(staged) != ""
	hasUnstagedDiff := strings.TrimSpace(unstaged) != ""

	if !hasStagedDiff && !hasUnstagedDiff {
		fmt.Fprintln(os.Stdout, tui.Dim("No changes found to review."))
		return nil
	}

	fmt.Fprintln(os.Stdout, tui.Bold("Code review — current changes"))
	fmt.Fprintln(os.Stdout)

	if hasStagedDiff {
		fmt.Fprintln(os.Stdout, tui.BoldUnderline("Staged changes:"))
		fmt.Fprintln(os.Stdout, colorizeDiff(staged))
		fmt.Fprintln(os.Stdout)
	}

	if hasUnstagedDiff {
		fmt.Fprintln(os.Stdout, tui.BoldUnderline("Unstaged changes:"))
		fmt.Fprintln(os.Stdout, colorizeDiff(unstaged))
		fmt.Fprintln(os.Stdout)
	}

	fmt.Fprintln(os.Stdout, tui.HorizontalRule())
	fmt.Fprintln(os.Stdout, tui.Dim("Paste the above diff into your next prompt to request an LLM code review."))
	return nil
}
