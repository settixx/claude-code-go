package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager orchestrates skill loading from multiple search paths and provides execution.
type Manager struct {
	registry    *SkillRegistry
	searchPaths []string
}

// NewManager creates a Manager that searches user home, project root, and bundled skills.
func NewManager(projectRoot string) *Manager {
	home, _ := os.UserHomeDir()
	var paths []string
	if home != "" {
		paths = append(paths, filepath.Join(home, ".claude", "skills"))
	}
	if projectRoot != "" {
		paths = append(paths, filepath.Join(projectRoot, ".claude", "skills"))
	}
	return &Manager{
		registry:    NewSkillRegistry(),
		searchPaths: paths,
	}
}

// Registry returns the underlying skill registry.
func (m *Manager) Registry() *SkillRegistry { return m.registry }

// LoadAll discovers and registers skills from all sources.
// Load order: bundled → user → project (later registrations override earlier ones).
func (m *Manager) LoadAll() error {
	bundled, err := LoadBundledSkills()
	if err != nil {
		return fmt.Errorf("load bundled skills: %w", err)
	}
	for _, s := range bundled {
		m.registry.Register(s)
	}

	sources := []string{"user", "project"}
	for i, dir := range m.searchPaths {
		source := "user"
		if i < len(sources) {
			source = sources[i]
		}
		skills, loadErr := LoadSkillsFromDir(dir, source)
		if loadErr != nil {
			return fmt.Errorf("load skills from %s: %w", dir, loadErr)
		}
		for _, s := range skills {
			m.registry.Register(s)
		}
	}
	return nil
}

// Reload clears the registry and reloads all skills from disk.
func (m *Manager) Reload() error {
	m.registry = NewSkillRegistry()
	return m.LoadAll()
}

// ExecuteSkill retrieves a skill's content by name, optionally prepending user args.
func (m *Manager) ExecuteSkill(name, args string) (string, error) {
	s, ok := m.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("skill %q not found", name)
	}
	if args == "" {
		return s.Content, nil
	}
	return s.Content + "\n\n## User Context\n\n" + args, nil
}
