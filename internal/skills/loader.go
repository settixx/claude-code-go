package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadSkillsFromDir recursively scans dir for SKILL.md files and returns parsed skills.
func LoadSkillsFromDir(dir, source string) ([]*Skill, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	var skills []*Skill
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(d.Name(), "SKILL.md") {
			return nil
		}
		s, parseErr := ParseSkillFile(path)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		s.Source = source
		s.FilePath = path
		skills = append(skills, s)
		return nil
	})
	if walkErr != nil {
		return skills, walkErr
	}
	return skills, nil
}

// ParseSkillFile reads a SKILL.md file, extracts YAML frontmatter and body content.
func ParseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseSkillContent(string(data), path)
}

// parseSkillContent parses skill markdown with optional YAML frontmatter.
func parseSkillContent(raw, filePath string) (*Skill, error) {
	s := &Skill{FilePath: filePath}

	frontmatter, body, hasFM := splitFrontmatter(raw)
	if hasFM {
		parseFrontmatter(frontmatter, s)
	}
	s.Content = strings.TrimSpace(body)

	if s.Name == "" {
		s.Name = inferNameFromPath(filePath)
	}
	return s, nil
}

// splitFrontmatter separates `---` delimited YAML frontmatter from body.
// Returns (frontmatter, body, found).
func splitFrontmatter(raw string) (string, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return "", raw, false
	}

	rest := trimmed[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", raw, false
	}

	fm := strings.TrimSpace(rest[:idx])
	body := rest[idx+4:] // skip "\n---"
	return fm, body, true
}

// parseFrontmatter extracts key-value pairs from simple YAML into a Skill.
func parseFrontmatter(fm string, s *Skill) {
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := parseKV(line)
		if !ok {
			continue
		}
		switch key {
		case "name":
			s.Name = val
		case "description":
			s.Description = val
		case "tags":
			s.Tags = parseYAMLList(val)
		case "user_invocable":
			s.UserInvoke = strings.EqualFold(val, "true")
		case "auto_run":
			s.AutoRun = strings.EqualFold(val, "true")
		case "source":
			s.Source = val
		}
	}
}

// parseKV splits "key: value" into (key, value, ok).
func parseKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return key, val, true
}

// parseYAMLList handles both `[a, b, c]` and multi-line `- item` styles.
func parseYAMLList(val string) []string {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		inner := val[1 : len(val)-1]
		parts := strings.Split(inner, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	if strings.HasPrefix(val, "-") {
		return []string{strings.TrimSpace(strings.TrimPrefix(val, "-"))}
	}
	if val != "" {
		return []string{val}
	}
	return nil
}

// inferNameFromPath derives a skill name from its file path.
// e.g. "/home/user/.claude/skills/git-commit/SKILL.md" → "git-commit"
func inferNameFromPath(path string) string {
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}
