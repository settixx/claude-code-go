package permissions

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// Rule defines a single permission matching rule with glob-based patterns.
type Rule struct {
	ToolPattern    string
	CommandPattern string
	Behavior       types.PermissionBehavior
	Session        bool // true = ephemeral session rule (not persisted)
}

// RuleSet holds categorized permission rules. Deny rules always take
// precedence over allow rules during evaluation. Session rules (added
// dynamically via AddAlwaysAllow / AddAlwaysDeny) are checked before
// file-based rules.
type RuleSet struct {
	mu         sync.RWMutex
	AllowRules []Rule
	DenyRules  []Rule
	AskRules   []Rule
}

// NewRuleSet returns an empty RuleSet ready for use.
func NewRuleSet() *RuleSet {
	return &RuleSet{}
}

// AddAllowRule appends a tool-name glob pattern as an allow rule.
func (rs *RuleSet) AddAllowRule(pattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.AllowRules = append(rs.AllowRules, Rule{
		ToolPattern: pattern,
		Behavior:    types.BehaviorAllow,
	})
}

// AddDenyRule appends a tool-name glob pattern as a deny rule.
func (rs *RuleSet) AddDenyRule(pattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.DenyRules = append(rs.DenyRules, Rule{
		ToolPattern: pattern,
		Behavior:    types.BehaviorDeny,
	})
}

// AddAskRule appends a tool-name glob pattern as an ask rule.
func (rs *RuleSet) AddAskRule(pattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.AskRules = append(rs.AskRules, Rule{
		ToolPattern: pattern,
		Behavior:    types.BehaviorAsk,
	})
}

// AddAllowCommandRule appends a rule that matches a specific tool pattern
// combined with a command glob (e.g. bash + "git *").
func (rs *RuleSet) AddAllowCommandRule(toolPattern, commandPattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.AllowRules = append(rs.AllowRules, Rule{
		ToolPattern:    toolPattern,
		CommandPattern: commandPattern,
		Behavior:       types.BehaviorAllow,
	})
}

// AddDenyCommandRule appends a deny rule with both tool and command patterns.
func (rs *RuleSet) AddDenyCommandRule(toolPattern, commandPattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.DenyRules = append(rs.DenyRules, Rule{
		ToolPattern:    toolPattern,
		CommandPattern: commandPattern,
		Behavior:       types.BehaviorDeny,
	})
}

// ---------- Session rules (ephemeral, not persisted) ----------

// AddAlwaysAllow adds a session-scoped allow rule for the given tool/pattern.
// Session rules take priority and are not persisted to disk.
func (rs *RuleSet) AddAlwaysAllow(toolName string, pattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	r := Rule{
		ToolPattern: toolName,
		Behavior:    types.BehaviorAllow,
		Session:     true,
	}
	if pattern != "" {
		r.CommandPattern = pattern
	}
	rs.AllowRules = append([]Rule{r}, rs.AllowRules...)
}

// AddAlwaysDeny adds a session-scoped deny rule for the given tool/pattern.
func (rs *RuleSet) AddAlwaysDeny(toolName string, pattern string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	r := Rule{
		ToolPattern: toolName,
		Behavior:    types.BehaviorDeny,
		Session:     true,
	}
	if pattern != "" {
		r.CommandPattern = pattern
	}
	rs.DenyRules = append([]Rule{r}, rs.DenyRules...)
}

// RemoveSessionRule removes all ephemeral session rules for the given tool.
func (rs *RuleSet) RemoveSessionRule(toolName string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.AllowRules = filterNonSession(rs.AllowRules, toolName)
	rs.DenyRules = filterNonSession(rs.DenyRules, toolName)
	rs.AskRules = filterNonSession(rs.AskRules, toolName)
}

// SessionRules returns all session-scoped rules currently active.
func (rs *RuleSet) SessionRules() []Rule {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var out []Rule
	for _, r := range rs.AllowRules {
		if r.Session {
			out = append(out, r)
		}
	}
	for _, r := range rs.DenyRules {
		if r.Session {
			out = append(out, r)
		}
	}
	return out
}

func filterNonSession(rules []Rule, toolName string) []Rule {
	filtered := rules[:0]
	for _, r := range rules {
		if r.Session && r.ToolPattern == toolName {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// ---------- Evaluate ----------

// Evaluate checks a tool invocation against all rules.
// Deny rules are checked first; if any match the result is deny.
// Then allow rules are checked; if any match the result is allow.
// If no rule matches, the result is ask.
func (rs *RuleSet) Evaluate(toolName string, input map[string]interface{}) types.PermissionBehavior {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	cmd := extractCommand(input)

	for _, r := range rs.DenyRules {
		if matchesRule(r, toolName, cmd) {
			return types.BehaviorDeny
		}
	}

	for _, r := range rs.AllowRules {
		if matchesRule(r, toolName, cmd) {
			return types.BehaviorAllow
		}
	}

	for _, r := range rs.AskRules {
		if matchesRule(r, toolName, cmd) {
			return types.BehaviorAsk
		}
	}

	return types.BehaviorAsk
}

func matchesRule(r Rule, toolName, command string) bool {
	toolMatched, _ := filepath.Match(r.ToolPattern, toolName)
	if !toolMatched {
		return false
	}
	if r.CommandPattern == "" {
		return true
	}
	if command == "" {
		return false
	}
	cmdMatched, _ := filepath.Match(r.CommandPattern, command)
	return cmdMatched
}

func extractCommand(input map[string]interface{}) string {
	for _, key := range []string{"command", "cmd", "script"} {
		if v, ok := input[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}
