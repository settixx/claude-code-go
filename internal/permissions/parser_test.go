package permissions

import (
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// ParsePermissionRules
// ---------------------------------------------------------------------------

func TestParsePermissionRules_Basic(t *testing.T) {
	content := `# Allowed Tools
- FileRead
- Glob
- Grep

# Denied Tools
- PowerShell
`
	rs, err := ParsePermissionRules(content)
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}

	if len(rs.AllowRules) != 3 {
		t.Errorf("AllowRules len = %d, want 3", len(rs.AllowRules))
	}
	if len(rs.DenyRules) != 1 {
		t.Errorf("DenyRules len = %d, want 1", len(rs.DenyRules))
	}
}

func TestParsePermissionRules_CompoundPattern(t *testing.T) {
	content := `# Allowed Tools
- Bash(git *)
- FileWrite(/tmp/*)
`
	rs, err := ParsePermissionRules(content)
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}

	if len(rs.AllowRules) != 2 {
		t.Fatalf("AllowRules len = %d, want 2", len(rs.AllowRules))
	}

	r := rs.AllowRules[0]
	if r.ToolPattern != "Bash" || r.CommandPattern != "git *" {
		t.Errorf("rule[0] = {%q, %q}, want {Bash, git *}", r.ToolPattern, r.CommandPattern)
	}

	r = rs.AllowRules[1]
	if r.ToolPattern != "FileWrite" || r.CommandPattern != "/tmp/*" {
		t.Errorf("rule[1] = {%q, %q}, want {FileWrite, /tmp/*}", r.ToolPattern, r.CommandPattern)
	}
}

func TestParsePermissionRules_EmptyContent(t *testing.T) {
	rs, err := ParsePermissionRules("")
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}
	if len(rs.AllowRules) != 0 || len(rs.DenyRules) != 0 {
		t.Error("expected empty rule set for empty content")
	}
}

func TestParsePermissionRules_NoSections(t *testing.T) {
	content := "Just some text\n- FileRead\n- Bash\n"
	rs, err := ParsePermissionRules(content)
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}
	if len(rs.AllowRules) != 0 || len(rs.DenyRules) != 0 {
		t.Error("expected no rules when no recognized section headings")
	}
}

func TestParsePermissionRules_IgnoresComments(t *testing.T) {
	// A "# comment" line is treated as a heading (since it starts with "#").
	// Headings that don't match "allow" or "deny" reset the current section,
	// so subsequent list items are not captured. This is expected behavior.
	content := `# Allowed Tools
- FileRead
# This is an unrelated heading
- Glob
`
	rs, err := ParsePermissionRules(content)
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}
	// Only FileRead is captured; Glob follows a non-allow/deny heading
	if len(rs.AllowRules) != 1 {
		t.Errorf("AllowRules len = %d, want 1 (Glob should be excluded by section reset)", len(rs.AllowRules))
	}
}

// ---------------------------------------------------------------------------
// parsePattern
// ---------------------------------------------------------------------------

func TestParsePattern(t *testing.T) {
	tests := []struct {
		input   string
		tool    string
		command string
	}{
		{"FileRead", "FileRead", ""},
		{"Bash(git *)", "Bash", "git *"},
		{"FileWrite(/tmp/*)", "FileWrite", "/tmp/*"},
		{"- Bash(ls -la)", "Bash", "ls -la"},
		{"NoParens", "NoParens", ""},
		{"UnclosedParen(missing", "UnclosedParen", "missing"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tool, cmd := parsePattern(tt.input)
			if tool != tt.tool || cmd != tt.command {
				t.Errorf("parsePattern(%q) = (%q, %q), want (%q, %q)",
					tt.input, tool, cmd, tt.tool, tt.command)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: ParsePermissionRules → RuleSet.Evaluate
// ---------------------------------------------------------------------------

func TestParsePermissionRules_FlowsIntoEvaluate(t *testing.T) {
	content := `# Allowed Tools
- FileRead
- Bash(git *)

# Denied Tools
- PowerShell
`
	rs, err := ParsePermissionRules(content)
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}

	tests := []struct {
		name  string
		tool  string
		input map[string]interface{}
		want  types.PermissionBehavior
	}{
		{"allowed plain", "FileRead", nil, types.BehaviorAllow},
		{"allowed command", "Bash", map[string]interface{}{"command": "git status"}, types.BehaviorAllow},
		{"bash non-git", "Bash", map[string]interface{}{"command": "rm -rf /"}, types.BehaviorAsk},
		{"denied tool", "PowerShell", nil, types.BehaviorDeny},
		{"unknown tool", "WebFetch", nil, types.BehaviorAsk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.Evaluate(tt.tool, tt.input)
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}
