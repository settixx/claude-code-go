package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed bundled/*.md
var bundledFS embed.FS

// LoadBundledSkills returns all skills embedded in the binary.
func LoadBundledSkills() ([]*Skill, error) {
	var skills []*Skill
	err := fs.WalkDir(bundledFS, "bundled", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, readErr := bundledFS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", path, readErr)
		}
		s, parseErr := parseSkillContent(string(data), path)
		if parseErr != nil {
			return fmt.Errorf("parse embedded %s: %w", path, parseErr)
		}
		s.Source = "bundled"
		skills = append(skills, s)
		return nil
	})
	return skills, err
}
