package config

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const maxIncludeDepth = 5

var includeRe = regexp.MustCompile(`^@include\s+(.+)$`)

// ClaudeMDRules holds the aggregated permission rules extracted from CLAUDE.md files.
type ClaudeMDRules struct {
	AllowPatterns []string
	DenyPatterns  []string
	Content       string
}

// LoadClaudeMD discovers CLAUDE.md files from multiple scopes (user-global,
// directory walk up to git root, project rules dirs, and CLAUDE.local.md),
// merges their rules, and strips HTML comments from the aggregated content.
func LoadClaudeMD(startDir string) (*ClaudeMDRules, error) {
	paths := discoverAllClaudeMDPaths(startDir)

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
	merged.Content = stripHTMLComments(strings.Join(sections, "\n\n"))
	return merged, nil
}

// discoverAllClaudeMDPaths collects CLAUDE.md paths from every scope:
//  1. User-global: ~/.claude/CLAUDE.md
//  2. User-global rules: ~/.claude/rules/*.md
//  3. Directory walk: startDir up to git root (deepest-first)
//  4. Project rules: startDir/.claude/rules/*.md
//  5. Local override: startDir/CLAUDE.local.md
func discoverAllClaudeMDPaths(startDir string) []string {
	var paths []string

	home := homeDir()
	paths = appendIfExists(paths, filepath.Join(home, ".claude", "CLAUDE.md"))
	paths = append(paths, collectMDsInDir(filepath.Join(home, ".claude", "rules"))...)

	root := findGitRoot(startDir)
	paths = append(paths, collectClaudeMDPaths(startDir, root)...)

	paths = append(paths, collectMDsInDir(filepath.Join(startDir, ".claude", "rules"))...)

	paths = appendIfExists(paths, filepath.Join(startDir, "CLAUDE.local.md"))

	return paths
}

// collectMDsInDir returns all *.md files in dir, sorted by name.
func collectMDsInDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	return paths
}

func appendIfExists(paths []string, path string) []string {
	if _, err := os.Stat(path); err == nil {
		return append(paths, path)
	}
	return paths
}

var htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)

// stripHTMLComments removes all <!-- ... --> blocks from s.
func stripHTMLComments(s string) string {
	return htmlCommentRe.ReplaceAllString(s, "")
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

// resolveIncludes processes @include directives in content, inlining referenced
// files. Supports up to maxIncludeDepth levels of recursion with cycle detection.
func resolveIncludes(content string, basePath string, depth int, visited map[string]bool) string {
	if depth >= maxIncludeDepth {
		return content
	}

	var result strings.Builder
	for _, line := range strings.Split(content, "\n") {
		m := includeRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			result.WriteString(line)
			result.WriteByte('\n')
			continue
		}

		includePath := resolveIncludePath(m[1], basePath)
		if includePath == "" || visited[includePath] {
			continue
		}

		data, err := os.ReadFile(includePath)
		if err != nil {
			log.Printf("config: @include %q: %v", m[1], err)
			continue
		}

		visited[includePath] = true
		resolved := resolveIncludes(string(data), filepath.Dir(includePath), depth+1, visited)
		result.WriteString(resolved)
		if !strings.HasSuffix(resolved, "\n") {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func resolveIncludePath(raw string, baseDir string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "\"'")
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(raw) {
		abs, err := filepath.Abs(raw)
		if err != nil {
			return ""
		}
		return abs
	}
	abs, err := filepath.Abs(filepath.Join(baseDir, raw))
	if err != nil {
		return ""
	}
	return abs
}

// parseClaudeMDFile reads a single CLAUDE.md and extracts allow/deny patterns.
// It resolves @include directives and checks YAML frontmatter conditions.
//
// Recognised line formats inside fenced sections:
//
//	# Allowed tools
//	- pattern
//
//	# Denied tools
//	- pattern
func parseClaudeMDFile(path string) (*ClaudeMDRules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	fm, content := ParseFrontmatter(content)
	if !fm.ShouldApply(filepath.Dir(path)) {
		return &ClaudeMDRules{}, nil
	}

	visited := map[string]bool{}
	absPath, _ := filepath.Abs(path)
	if absPath != "" {
		visited[absPath] = true
	}
	content = resolveIncludes(content, filepath.Dir(path), 0, visited)

	return parseClaudeMDContent(content), nil
}

// parseClaudeMDContent extracts rules from already-resolved CLAUDE.md content.
func parseClaudeMDContent(content string) *ClaudeMDRules {
	rules := &ClaudeMDRules{}
	var contentBuf strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))

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
	return rules
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
