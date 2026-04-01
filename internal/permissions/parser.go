package permissions

import (
	"strings"
)

// ParsePermissionRules extracts permission rules from CLAUDE.md content.
//
// Recognised section headings (case-insensitive):
//
//	# Allowed Tools   → allow rules
//	# Denied Tools    → deny rules
//
// Each non-empty line under a heading is treated as a pattern.
// Plain tool names are matched directly (e.g. "FileRead").
// Compound patterns like "Bash(git *)" expand to a tool pattern + command pattern.
// Path-scoped patterns like "FileWrite(/tmp/*)" expand similarly.
func ParsePermissionRules(content string) (*RuleSet, error) {
	rs := NewRuleSet()
	lines := strings.Split(content, "\n")

	var section string // "allow" | "deny" | ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		if isHeading(line) {
			section = classifyHeading(line)
			continue
		}

		if section == "" || line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		toolPattern, cmdPattern := parsePattern(line)
		addParsedRule(rs, section, toolPattern, cmdPattern)
	}

	return rs, nil
}

func isHeading(line string) bool {
	return strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ")
}

func classifyHeading(line string) string {
	lower := strings.ToLower(line)
	lower = strings.TrimLeft(lower, "# ")

	switch {
	case strings.Contains(lower, "allowed") || strings.Contains(lower, "allow"):
		return "allow"
	case strings.Contains(lower, "denied") || strings.Contains(lower, "deny"):
		return "deny"
	default:
		return ""
	}
}

// parsePattern splits "Bash(git *)" into ("Bash", "git *") or returns
// ("FileRead", "") for plain names. Also handles list-item prefixes.
func parsePattern(line string) (string, string) {
	line = strings.TrimLeft(line, "- ")
	line = strings.TrimSpace(line)

	idx := strings.Index(line, "(")
	if idx < 0 {
		return line, ""
	}

	toolPart := strings.TrimSpace(line[:idx])
	rest := line[idx+1:]
	end := strings.LastIndex(rest, ")")
	if end < 0 {
		return toolPart, strings.TrimSpace(rest)
	}
	return toolPart, strings.TrimSpace(rest[:end])
}

func addParsedRule(rs *RuleSet, section, toolPattern, cmdPattern string) {
	switch section {
	case "allow":
		if cmdPattern != "" {
			rs.AddAllowCommandRule(toolPattern, cmdPattern)
		} else {
			rs.AddAllowRule(toolPattern)
		}
	case "deny":
		if cmdPattern != "" {
			rs.AddDenyCommandRule(toolPattern, cmdPattern)
		} else {
			rs.AddDenyRule(toolPattern)
		}
	}
}
