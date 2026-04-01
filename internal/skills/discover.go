package skills

import "strings"

// Search performs a simple keyword search across skill names, descriptions, and tags.
func (r *SkillRegistry) Search(query string) []*Skill {
	if query == "" {
		return r.All()
	}

	q := strings.ToLower(query)
	terms := strings.Fields(q)

	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*Skill
	for _, s := range r.skills {
		if matchesAllTerms(s, terms) {
			out = append(out, s)
		}
	}
	return out
}

// matchesAllTerms returns true if every query term appears in the skill's searchable text.
func matchesAllTerms(s *Skill, terms []string) bool {
	searchable := buildSearchText(s)
	for _, t := range terms {
		if !strings.Contains(searchable, t) {
			return false
		}
	}
	return true
}

func buildSearchText(s *Skill) string {
	parts := []string{
		strings.ToLower(s.Name),
		strings.ToLower(s.Description),
	}
	for _, tag := range s.Tags {
		parts = append(parts, strings.ToLower(tag))
	}
	return strings.Join(parts, " ")
}
