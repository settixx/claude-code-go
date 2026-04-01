package skills

import "sync"

// Skill represents a loaded skill definition with its metadata and prompt content.
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	UserInvoke  bool     `json:"user_invocable"`
	AutoRun     bool     `json:"auto_run"`
	Source      string   `json:"source"`    // "bundled", "user", "project"
	FilePath    string   `json:"file_path"` // Absolute path to SKILL.md on disk
}

// SkillRegistry is a thread-safe container of named skills.
type SkillRegistry struct {
	mu     sync.RWMutex
	skills map[string]*Skill
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{skills: make(map[string]*Skill)}
}

// Register adds or replaces a skill in the registry.
func (r *SkillRegistry) Register(s *Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[s.Name] = s
}

// Get returns the skill with the given name, if any.
func (r *SkillRegistry) Get(name string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[name]
	return s, ok
}

// All returns every registered skill.
func (r *SkillRegistry) All() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Skill, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, s)
	}
	return out
}

// UserInvocable returns skills that the user can invoke directly.
func (r *SkillRegistry) UserInvocable() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Skill
	for _, s := range r.skills {
		if s.UserInvoke {
			out = append(out, s)
		}
	}
	return out
}

// FindByTag returns skills matching the given tag.
func (r *SkillRegistry) FindByTag(tag string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Skill
	for _, s := range r.skills {
		for _, t := range s.Tags {
			if t == tag {
				out = append(out, s)
				break
			}
		}
	}
	return out
}
