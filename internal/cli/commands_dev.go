package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/settixx/claude-code-go/internal/mcp"
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
		Name:        "doctor",
		Description: "Run environment diagnostics",
		Handler:     cmdDoctor,
	})
}

func cmdCommit(_ string, ctx *CommandContext) error {
	cwd := effectiveCWD(ctx)

	stagedStat, err := runGit(cwd, "diff", "--cached", "--stat")
	if err != nil {
		return fmt.Errorf("git diff --cached: %w", err)
	}

	if strings.TrimSpace(stagedStat) == "" {
		return suggestStaging(cwd)
	}

	commitMsg := generateCommitMessage(stagedStat)

	fmt.Fprintln(os.Stdout, tui.Bold("Staged changes:"))
	fmt.Fprintln(os.Stdout, tui.Dim(stagedStat))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Proposed commit message: %s\n", tui.Cyan(commitMsg))
	fmt.Fprintln(os.Stdout)

	out, err := runGit(cwd, "commit", "-m", commitMsg)
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	fmt.Fprintln(os.Stdout, tui.Green("✓ Committed:"))
	fmt.Fprintln(os.Stdout, out)
	return nil
}

func suggestStaging(cwd string) error {
	unstagedStat, err := runGit(cwd, "diff", "--stat")
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	if strings.TrimSpace(unstagedStat) == "" {
		fmt.Fprintln(os.Stdout, tui.Dim("No staged or unstaged changes found."))
		return nil
	}

	fmt.Fprintln(os.Stdout, tui.Yellow("No staged changes. Unstaged changes:"))
	fmt.Fprintln(os.Stdout, tui.Dim(unstagedStat))
	fmt.Fprintln(os.Stdout, tui.Dim("Stage changes with: git add <files>"))
	return nil
}

func generateCommitMessage(stat string) string {
	lines := strings.Split(strings.TrimSpace(stat), "\n")
	if len(lines) == 0 {
		return "chore: update files"
	}

	summaryLine := strings.TrimSpace(lines[0])
	if idx := strings.Index(summaryLine, "|"); idx > 0 {
		summaryLine = strings.TrimSpace(summaryLine[:idx])
	}

	fileCount := 0
	for _, l := range lines {
		if strings.Contains(l, "|") {
			fileCount++
		}
	}

	if fileCount == 1 {
		return fmt.Sprintf("feat: update %s", summaryLine)
	}
	return fmt.Sprintf("feat: update %d files (%s)", fileCount, summaryLine)
}

func cmdDiff(args string, ctx *CommandContext) error {
	cwd := effectiveCWD(ctx)

	gitArgs := []string{"diff"}
	if strings.Contains(args, "--cached") || strings.Contains(args, "--staged") {
		gitArgs = append(gitArgs, "--cached")
	}

	out, err := runGit(cwd, gitArgs...)
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		label := "working tree"
		if len(gitArgs) > 1 {
			label = "staged"
		}
		fmt.Fprintf(os.Stdout, "%s\n", tui.Dim("No "+label+" changes."))
		return nil
	}

	fmt.Fprintln(os.Stdout, colorizeDiff(out))
	return nil
}

func colorizeDiff(raw string) string {
	var b strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			b.WriteString(tui.Bold(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(tui.Cyan(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(tui.Green(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(tui.Red(line))
		case strings.HasPrefix(line, "diff "):
			b.WriteString(tui.Bold(line))
		default:
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func cmdDoctor(_ string, ctx *CommandContext) error {
	fmt.Fprintln(os.Stdout, tui.Bold("Doctor — environment diagnostics"))
	fmt.Fprintln(os.Stdout)

	checks := []struct {
		name string
		fn   func(*CommandContext) (string, bool)
	}{
		{"Go version", checkGoVersion},
		{"Git", checkGit},
		{"API key", checkAPIKey},
		{"MCP servers", checkMCPServers},
		{"CWD writable", checkCWDWritable},
	}

	for _, c := range checks {
		detail, ok := c.fn(ctx)
		icon := tui.Green("✓")
		if !ok {
			icon = tui.Red("✗")
		}
		fmt.Fprintf(os.Stdout, "  %s %-18s %s\n", icon, c.name, tui.Dim(detail))
	}
	fmt.Fprintln(os.Stdout)
	return nil
}

func checkGoVersion(_ *CommandContext) (string, bool) {
	return runtime.Version(), true
}

func checkGit(_ *CommandContext) (string, bool) {
	out, err := exec.Command("git", "--version").CombinedOutput()
	if err != nil {
		return "not found", false
	}
	return strings.TrimSpace(string(out)), true
}

func checkAPIKey(_ *CommandContext) (string, bool) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return "ANTHROPIC_API_KEY not set", false
	}
	masked := key[:4] + "..." + key[len(key)-4:]
	return masked, true
}

func checkMCPServers(_ *CommandContext) (string, bool) {
	cfgPath, err := mcp.DefaultConfigPath()
	if err != nil {
		return "config path error", false
	}
	servers, err := mcp.LoadMCPConfig(cfgPath)
	if err != nil {
		return fmt.Sprintf("config error: %v", err), false
	}
	if len(servers) == 0 {
		return "no servers configured", true
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	return fmt.Sprintf("%d server(s): %s", len(servers), strings.Join(names, ", ")), true
}

func checkCWDWritable(ctx *CommandContext) (string, bool) {
	dir := effectiveCWD(ctx)
	tmp := dir + "/.ti-code-doctor-probe"
	if err := os.WriteFile(tmp, []byte("probe"), 0o644); err != nil {
		return fmt.Sprintf("%s — not writable", dir), false
	}
	os.Remove(tmp)
	return dir, true
}

func effectiveCWD(ctx *CommandContext) string {
	if ctx.CWD != "" {
		return ctx.CWD
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func runGit(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
