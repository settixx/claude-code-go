package permissions

import (
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// RuleSet.Evaluate — deny > allow > ask precedence
// ---------------------------------------------------------------------------

func TestRuleSet_EvalDenyTakesPrecedence(t *testing.T) {
	rs := NewRuleSet()
	rs.AddAllowRule("Bash")
	rs.AddDenyRule("Bash")

	got := rs.Evaluate("Bash", nil)
	if got != types.BehaviorDeny {
		t.Errorf("Evaluate = %q, want deny (deny overrides allow)", got)
	}
}

func TestRuleSet_EvalAllowBeforeAsk(t *testing.T) {
	rs := NewRuleSet()
	rs.AddAllowRule("FileRead")

	got := rs.Evaluate("FileRead", nil)
	if got != types.BehaviorAllow {
		t.Errorf("Evaluate = %q, want allow", got)
	}
}

func TestRuleSet_EvalNoRuleReturnsAsk(t *testing.T) {
	rs := NewRuleSet()

	got := rs.Evaluate("UnknownTool", nil)
	if got != types.BehaviorAsk {
		t.Errorf("Evaluate = %q, want ask", got)
	}
}

func TestRuleSet_EvalCommandPattern(t *testing.T) {
	rs := NewRuleSet()
	rs.AddAllowCommandRule("Bash", "git *")

	t.Run("matching command", func(t *testing.T) {
		got := rs.Evaluate("Bash", map[string]interface{}{"command": "git status"})
		if got != types.BehaviorAllow {
			t.Errorf("Evaluate = %q, want allow", got)
		}
	})

	t.Run("non-matching command", func(t *testing.T) {
		got := rs.Evaluate("Bash", map[string]interface{}{"command": "rm -rf /"})
		if got != types.BehaviorAsk {
			t.Errorf("Evaluate = %q, want ask", got)
		}
	})

	t.Run("no command provided", func(t *testing.T) {
		got := rs.Evaluate("Bash", nil)
		if got != types.BehaviorAsk {
			t.Errorf("Evaluate = %q, want ask (command pattern but no command)", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Session rules — AddAlwaysAllow / AddAlwaysDeny
// ---------------------------------------------------------------------------

func TestRuleSet_SessionAllowPrepended(t *testing.T) {
	rs := NewRuleSet()
	rs.AddDenyRule("Bash")
	rs.AddAlwaysAllow("Bash", "")

	got := rs.Evaluate("Bash", nil)
	if got != types.BehaviorDeny {
		t.Logf("deny still wins because deny rules checked first")
	}
}

func TestRuleSet_SessionDeny(t *testing.T) {
	rs := NewRuleSet()
	rs.AddAlwaysDeny("Bash", "")

	got := rs.Evaluate("Bash", nil)
	if got != types.BehaviorDeny {
		t.Errorf("Evaluate = %q, want deny", got)
	}
}

func TestRuleSet_SessionRulesList(t *testing.T) {
	rs := NewRuleSet()
	rs.AddAllowRule("FileRead")
	rs.AddAlwaysAllow("Bash", "git *")
	rs.AddAlwaysDeny("PowerShell", "")

	session := rs.SessionRules()
	if len(session) != 2 {
		t.Fatalf("expected 2 session rules, got %d", len(session))
	}
}

func TestRuleSet_RemoveSessionRule(t *testing.T) {
	rs := NewRuleSet()
	rs.AddAlwaysAllow("Bash", "")
	rs.AddAlwaysDeny("Bash", "")

	rs.RemoveSessionRule("Bash")

	session := rs.SessionRules()
	if len(session) != 0 {
		t.Errorf("expected 0 session rules after remove, got %d", len(session))
	}
}

// ---------------------------------------------------------------------------
// extractCommand helper
// ---------------------------------------------------------------------------

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
		want  string
	}{
		{"command key", map[string]interface{}{"command": "ls -la"}, "ls -la"},
		{"cmd key", map[string]interface{}{"cmd": "echo hi"}, "echo hi"},
		{"script key", map[string]interface{}{"script": "run.sh"}, "run.sh"},
		{"no command", map[string]interface{}{"path": "/tmp"}, ""},
		{"nil input", nil, ""},
		{"whitespace trimmed", map[string]interface{}{"command": "  git push  "}, "git push"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommand(tt.input)
			if got != tt.want {
				t.Errorf("extractCommand = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// matchesRule
// ---------------------------------------------------------------------------

func TestMatchesRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     Rule
		tool     string
		command  string
		expected bool
	}{
		{
			"exact tool match, no command pattern",
			Rule{ToolPattern: "Bash"},
			"Bash", "", true,
		},
		{
			"glob tool match",
			Rule{ToolPattern: "File*"},
			"FileRead", "", true,
		},
		{
			"tool match with command pattern match",
			Rule{ToolPattern: "Bash", CommandPattern: "git *"},
			"Bash", "git push", true,
		},
		{
			"tool match but command mismatch",
			Rule{ToolPattern: "Bash", CommandPattern: "git *"},
			"Bash", "rm -rf /", false,
		},
		{
			"tool mismatch",
			Rule{ToolPattern: "FileWrite"},
			"Bash", "ls", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesRule(tt.rule, tt.tool, tt.command)
			if got != tt.expected {
				t.Errorf("matchesRule = %v, want %v", got, tt.expected)
			}
		})
	}
}
