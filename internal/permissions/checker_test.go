package permissions

import (
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

func TestBypassModeAllowsEverything(t *testing.T) {
	checker := NewChecker(types.PermBypassPermissions, nil)

	tools := []string{"Bash", "FileWrite", "FileEdit", "FileRead", "Glob"}
	for _, name := range tools {
		t.Run(name, func(t *testing.T) {
			result := checker.Check(name, nil)
			if !result.Allowed {
				t.Errorf("bypass mode denied %q: %s", name, result.Reason)
			}
		})
	}
}

func TestDontAskModeAllowsEverything(t *testing.T) {
	checker := NewChecker(types.PermDontAsk, nil)

	result := checker.Check("Bash", map[string]interface{}{"command": "rm -rf /"})
	if !result.Allowed {
		t.Errorf("dontAsk mode denied: %s", result.Reason)
	}
}

func TestPlanModeDeniesWrites(t *testing.T) {
	checker := NewChecker(types.PermPlan, nil)

	writeTools := []string{"Bash", "FileWrite", "FileEdit", "NotebookEdit"}
	for _, name := range writeTools {
		t.Run("deny_"+name, func(t *testing.T) {
			result := checker.Check(name, nil)
			if result.Allowed {
				t.Errorf("plan mode should deny %q", name)
			}
		})
	}

	readTools := []string{"FileRead", "Glob", "Grep"}
	for _, name := range readTools {
		t.Run("allow_"+name, func(t *testing.T) {
			result := checker.Check(name, nil)
			if !result.Allowed {
				t.Errorf("plan mode should allow read tool %q: %s", name, result.Reason)
			}
		})
	}
}

func TestDefaultModeWithRules(t *testing.T) {
	rules := NewRuleSet()
	rules.AddAllowRule("FileRead")
	rules.AddDenyRule("Bash")

	checker := NewChecker(types.PermDefault, rules)

	t.Run("allowed by rule", func(t *testing.T) {
		result := checker.Check("FileRead", nil)
		if !result.Allowed {
			t.Errorf("FileRead should be allowed by rule: %s", result.Reason)
		}
	})

	t.Run("denied by rule", func(t *testing.T) {
		result := checker.Check("Bash", nil)
		if result.Allowed {
			t.Error("Bash should be denied by rule")
		}
	})

	t.Run("no rule requires confirmation", func(t *testing.T) {
		result := checker.Check("SomeUnknownTool", nil)
		if result.Allowed {
			t.Error("unknown tool with no rule should not be auto-allowed")
		}
	})
}

func TestSetMode(t *testing.T) {
	checker := NewChecker(types.PermDefault, nil)

	if checker.Mode() != types.PermDefault {
		t.Errorf("initial mode = %q, want %q", checker.Mode(), types.PermDefault)
	}

	checker.SetMode(types.PermBypassPermissions)
	if checker.Mode() != types.PermBypassPermissions {
		t.Errorf("mode after set = %q, want %q", checker.Mode(), types.PermBypassPermissions)
	}

	result := checker.Check("Bash", nil)
	if !result.Allowed {
		t.Error("after switching to bypass, Bash should be allowed")
	}
}

func TestAcceptEditsMode(t *testing.T) {
	checker := NewChecker(types.PermAcceptEdits, nil)

	t.Run("read-only tool allowed", func(t *testing.T) {
		result := checker.Check("FileRead", nil)
		if !result.Allowed {
			t.Errorf("FileRead should be allowed: %s", result.Reason)
		}
	})

	t.Run("file edit tool allowed", func(t *testing.T) {
		result := checker.Check("FileEdit", nil)
		if !result.Allowed {
			t.Errorf("FileEdit should be allowed in acceptEdits: %s", result.Reason)
		}
	})

	t.Run("safe bash allowed", func(t *testing.T) {
		result := checker.Check("Bash", map[string]interface{}{"command": "ls -la"})
		if !result.Allowed {
			t.Errorf("safe bash should be allowed in acceptEdits: %s", result.Reason)
		}
	})
}

func TestAutoModeWithRules(t *testing.T) {
	rules := NewRuleSet()
	rules.AddAllowRule("FileRead")
	rules.AddDenyRule("Bash")

	checker := NewChecker(types.PermAuto, rules)

	t.Run("allowed by explicit rule", func(t *testing.T) {
		result := checker.Check("FileRead", nil)
		if !result.Allowed {
			t.Errorf("FileRead should be allowed: %s", result.Reason)
		}
	})

	t.Run("denied by explicit rule", func(t *testing.T) {
		result := checker.Check("Bash", nil)
		if result.Allowed {
			t.Error("Bash should be denied by rule")
		}
	})
}

// ---------------------------------------------------------------------------
// ClaudeMD → ParsePermissionRules → Checker integration
// ---------------------------------------------------------------------------

func TestClaudeMDRulesFlowIntoChecker(t *testing.T) {
	content := `# Allowed Tools
- FileRead
- Bash(git *)

# Denied Tools
- PowerShell
- WebFetch
`
	rs, err := ParsePermissionRules(content)
	if err != nil {
		t.Fatalf("ParsePermissionRules: %v", err)
	}

	checker := NewChecker(types.PermDefault, rs)

	tests := []struct {
		name    string
		tool    string
		input   map[string]interface{}
		allowed bool
	}{
		{"allowed FileRead", "FileRead", nil, true},
		{"allowed git command", "Bash", map[string]interface{}{"command": "git status"}, true},
		{"denied PowerShell", "PowerShell", nil, false},
		{"denied WebFetch", "WebFetch", nil, false},
		{"unknown tool needs confirmation", "Agent", nil, false},
		{"bash non-git needs confirmation", "Bash", map[string]interface{}{"command": "rm -rf /"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.Check(tt.tool, tt.input)
			if result.Allowed != tt.allowed {
				t.Errorf("Check(%q) allowed=%v, want %v (reason: %s)", tt.tool, result.Allowed, tt.allowed, result.Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Checker — decision history
// ---------------------------------------------------------------------------

func TestCheckerDecisionHistory(t *testing.T) {
	rules := NewRuleSet()
	rules.AddAllowRule("FileRead")
	rules.AddDenyRule("Bash")

	checker := NewChecker(types.PermDefault, rules)

	checker.Check("FileRead", nil)
	checker.Check("Bash", nil)

	history := checker.DecisionHistory()
	if len(history) != 0 {
		t.Errorf("Check() alone should not record history, got %d entries", len(history))
	}
}

// ---------------------------------------------------------------------------
// Checker — BuildRequest
// ---------------------------------------------------------------------------

func TestCheckerBuildRequest(t *testing.T) {
	checker := NewChecker(types.PermDefault, nil)

	t.Run("bash command", func(t *testing.T) {
		req := checker.BuildRequest("Bash", map[string]interface{}{"command": "ls -la"})
		if req.ToolName != "Bash" {
			t.Errorf("ToolName = %q", req.ToolName)
		}
		if req.Description == "" {
			t.Error("Description should not be empty")
		}
	})

	t.Run("file tool", func(t *testing.T) {
		req := checker.BuildRequest("FileWrite", map[string]interface{}{"file_path": "/tmp/x"})
		if req.ToolName != "FileWrite" {
			t.Errorf("ToolName = %q", req.ToolName)
		}
	})

	t.Run("no input", func(t *testing.T) {
		req := checker.BuildRequest("Agent", nil)
		if req.Description == "" {
			t.Error("Description should not be empty even with nil input")
		}
	})
}

// ---------------------------------------------------------------------------
// Checker — command pattern rules
// ---------------------------------------------------------------------------

func TestCheckerCommandPatternRules(t *testing.T) {
	rules := NewRuleSet()
	rules.AddAllowCommandRule("Bash", "git *")
	rules.AddDenyCommandRule("Bash", "rm *")

	checker := NewChecker(types.PermDefault, rules)

	tests := []struct {
		name    string
		cmd     string
		allowed bool
	}{
		{"git allowed", "git status", true},
		{"rm denied", "rm -rf /", false},
		{"other needs confirmation", "ls -la", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.Check("Bash", map[string]interface{}{"command": tt.cmd})
			if result.Allowed != tt.allowed {
				t.Errorf("cmd=%q: allowed=%v, want %v (reason: %s)", tt.cmd, result.Allowed, tt.allowed, result.Reason)
			}
		})
	}
}
