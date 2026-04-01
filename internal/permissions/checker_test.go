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
