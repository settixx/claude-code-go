package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitInfo holds cached git repository information.
type GitInfo struct {
	IsRepo        bool
	Branch        string
	HasUnpushed   bool
	UnpushedCount int
}

// DetectGit gathers git information for the given directory.
func DetectGit(dir string) GitInfo {
	info := GitInfo{}

	if !isInsideWorkTree(dir) {
		return info
	}
	info.IsRepo = true
	info.Branch = currentBranch(dir)
	info.UnpushedCount = unpushedCount(dir)
	info.HasUnpushed = info.UnpushedCount > 0
	return info
}

// GenerateCommitMessages generates candidate commit messages from the current diff.
func GenerateCommitMessages(dir string) ([]string, error) {
	stat := gitOutput(dir, "diff", "--cached", "--stat")
	if stat == "" {
		return nil, fmt.Errorf("no staged changes found")
	}

	nameOut := gitOutput(dir, "diff", "--cached", "--name-only")
	if nameOut == "" {
		return nil, fmt.Errorf("no staged files found")
	}

	files := strings.Split(nameOut, "\n")
	return generateBasicCandidates(files), nil
}

func isInsideWorkTree(dir string) bool {
	out := gitOutput(dir, "rev-parse", "--is-inside-work-tree")
	return out == "true"
}

func currentBranch(dir string) string {
	return gitOutput(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

func unpushedCount(dir string) int {
	out := gitOutput(dir, "log", "@{u}..", "--oneline")
	if out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

// gitOutput runs a git command and returns its trimmed stdout, or "" on error.
func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func generateBasicCandidates(files []string) []string {
	if len(files) == 0 {
		return []string{"chore: update files"}
	}
	if len(files) == 1 {
		base := filepath.Base(files[0])
		return []string{
			fmt.Sprintf("update %s", base),
			fmt.Sprintf("fix: update %s", base),
			fmt.Sprintf("feat: update %s", base),
		}
	}

	dir := commonDir(files)
	return []string{
		fmt.Sprintf("update %d files in %s", len(files), dir),
		fmt.Sprintf("fix: update %s", dir),
		fmt.Sprintf("feat: update %s", dir),
	}
}

func commonDir(files []string) string {
	if len(files) == 0 {
		return "."
	}
	parts := strings.Split(files[0], "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return "."
}
