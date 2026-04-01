package coordinator

import (
	"fmt"
	"os/exec"
	"strings"
)

// WorktreeInfo describes one git worktree entry.
type WorktreeInfo struct {
	Path   string
	Branch string
	HEAD   string
}

// CreateWorktree runs `git worktree add` to create an isolated working
// directory on the given branch. Returns the absolute path of the new worktree.
func CreateWorktree(baseDir string, branchName string) (string, error) {
	worktreePath := baseDir + "/../" + branchName
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = baseDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return resolveAbsPath(worktreePath), nil
}

// RemoveWorktree runs `git worktree remove` to clean up a worktree.
func RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// ListWorktrees returns all worktrees known to the repository at baseDir.
func ListWorktrees(baseDir string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = baseDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return parseWorktreeList(string(out)), nil
}

func parseWorktreeList(raw string) []WorktreeInfo {
	var result []WorktreeInfo
	var current WorktreeInfo
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				result = append(result, current)
			}
			current = WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch ")
		}
	}
	if current.Path != "" {
		result = append(result, current)
	}
	return result
}

func resolveAbsPath(path string) string {
	cmd := exec.Command("realpath", path)
	out, err := cmd.Output()
	if err != nil {
		return path
	}
	return strings.TrimSpace(string(out))
}
