package config

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ClaudeMDRules holds the aggregated permission rules extracted from CLAUDE.md files.
type ClaudeMDRules struct {
	AllowPatterns []string
	DenyPatterns  []string
	Content       string
}

// LoadClaudeMD searches upward from startDir to the git root (or filesystem root)
// for CLAUDE.md files. Rules from closer directories take precedence.
// Returns the merged rules from all discovered files.
func LoadClaudeMD(startDir string) (*ClaudeMDRules, error) {
	root := findGitRoot(startDir)
	paths := collectClaudeMDPaths(startDir, root)

	merged := &ClaudeMDRules{}
	var sections []string
	for _, p := range paths {
		rules, err := parseClaudeMDFile(p)
		if err != nil {
			continue
		}
		merged.AllowPatterns = append(merged.AllowPatterns, rules.AllowPatterns...)
		merged.DenyPatterns = append(merged.DenyPatterns, rules.DenyPatterns...)
		if rules.Content != "" {
			sections = append(sections, rules.Content)
		}
	}
	merged.Content = strings.Join(sections, "\n\n")
	return merged, nil
}

// collectClaudeMDPaths walks from startDir up to root, collecting every
// CLAUDE.md it finds. Paths are returned deepest-first (highest precedence first).
func collectClaudeMDPaths(startDir, root string) []string {
	var paths []string
	dir := startDir
	for {
		candidate := filepath.Join(dir, "CLAUDE.md")
		if _, err := os.Stat(candidate); err == nil {
			paths = append(paths, candidate)
		}
		if dir == root || dir == "/" || dir == "." {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return paths
}

// parseClaudeMDFile reads a single CLAUDE.md and extracts allow/deny patterns.
//
// Recognised line formats inside fenced sections:
//
//	# Allowed tools
//	- pattern
//
//	# Denied tools
//	- pattern
func parseClaudeMDFile(path string) (*ClaudeMDRules, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rules := &ClaudeMDRules{}
	var contentBuf strings.Builder
	scanner := bufio.NewScanner(f)

	const (
		sectionNone  = ""
		sectionAllow = "allow"
		sectionDeny  = "deny"
	)
	section := sectionNone

	for scanner.Scan() {
		line := scanner.Text()
		contentBuf.WriteString(line)
		contentBuf.WriteByte('\n')

		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if isHeading(trimmed) {
			section = classifyHeading(lower)
			continue
		}

		pattern := extractListItem(trimmed)
		if pattern == "" {
			continue
		}

		switch section {
		case sectionAllow:
			rules.AllowPatterns = append(rules.AllowPatterns, pattern)
		case sectionDeny:
			rules.DenyPatterns = append(rules.DenyPatterns, pattern)
		}
	}

	rules.Content = strings.TrimSpace(contentBuf.String())
	return rules, scanner.Err()
}

func isHeading(line string) bool {
	return strings.HasPrefix(line, "#")
}

func classifyHeading(lower string) string {
	stripped := strings.TrimLeft(lower, "# ")
	switch {
	case strings.Contains(stripped, "allowed") || strings.Contains(stripped, "allow"):
		return "allow"
	case strings.Contains(stripped, "denied") || strings.Contains(stripped, "deny"):
		return "deny"
	default:
		return ""
	}
}

func extractListItem(line string) string {
	if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
		return ""
	}
	item := strings.TrimSpace(line[2:])
	item = strings.Trim(item, "`")
	return item
}

// findGitRoot returns the git repository root for dir, or dir itself if
// not inside a repo.
func findGitRoot(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return dir
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return dir
	}
	return root
}
