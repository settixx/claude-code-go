package config

import (
	"path/filepath"
	"strings"
)

// Frontmatter holds parsed YAML-like frontmatter conditions.
type Frontmatter struct {
	Paths []string
}

// ParseFrontmatter extracts frontmatter from content and returns the
// frontmatter + remaining content. Returns nil frontmatter if none found.
func ParseFrontmatter(content string) (*Frontmatter, string) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, content
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return nil, content
	}
	fmBlock := content[4 : 4+end]
	remaining := content[4+end+4:]

	fm := &Frontmatter{}
	for _, line := range strings.Split(fmBlock, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		pattern := strings.Trim(line[2:], " \"'")
		fm.Paths = append(fm.Paths, pattern)
	}
	return fm, remaining
}

// ShouldApply checks if frontmatter conditions are met for the given cwd.
// A nil Frontmatter or empty Paths list always applies.
func (f *Frontmatter) ShouldApply(cwd string) bool {
	if f == nil || len(f.Paths) == 0 {
		return true
	}
	for _, pattern := range f.Paths {
		matched, _ := filepath.Match(pattern, cwd)
		if matched {
			return true
		}
		if !strings.Contains(pattern, "**") {
			continue
		}
		base := strings.Split(pattern, "**")[0]
		if strings.Contains(cwd, strings.TrimSuffix(base, "/")) {
			return true
		}
	}
	return false
}
