package agent

import (
	"strings"
	"testing"

	"github.com/settixx/claude-code-go/internal/coordinator"
)

// ---------------------------------------------------------------------------
// Agent Definitions
// ---------------------------------------------------------------------------

func TestGetBuiltInAgents_AllFiveExist(t *testing.T) {
	agents := GetBuiltInAgents()
	if len(agents) != 5 {
		t.Fatalf("expected 5 built-in agents, got %d", len(agents))
	}

	expected := map[string]bool{
		"explore": false, "plan": false, "code-reviewer": false,
		"verification": false, "generalPurpose": false,
	}
	for _, a := range agents {
		if _, ok := expected[a.AgentType]; !ok {
			t.Errorf("unexpected agent type %q", a.AgentType)
		}
		expected[a.AgentType] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing agent type %q", name)
		}
	}
}

func TestGetBuiltInAgents_ReturnsCopy(t *testing.T) {
	a := GetBuiltInAgents()
	a[0].AgentType = "tampered"
	b := GetBuiltInAgents()
	if b[0].AgentType == "tampered" {
		t.Error("GetBuiltInAgents should return a copy, not a reference")
	}
}

func TestFindAgent_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"explore", "explore"},
		{"EXPLORE", "explore"},
		{"Explore", "explore"},
		{"generalPurpose", "generalPurpose"},
		{"GENERALPURPOSE", "generalPurpose"},
		{"code-reviewer", "code-reviewer"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			def := FindAgent(tt.input)
			if def == nil {
				t.Fatalf("FindAgent(%q) returned nil", tt.input)
			}
			if def.AgentType != tt.want {
				t.Errorf("AgentType = %q, want %q", def.AgentType, tt.want)
			}
		})
	}
}

func TestFindAgent_NotFound(t *testing.T) {
	def := FindAgent("nonexistent")
	if def != nil {
		t.Error("expected nil for nonexistent agent type")
	}
}

// ---------------------------------------------------------------------------
// IsToolAllowed
// ---------------------------------------------------------------------------

func TestIsToolAllowed_AllowList(t *testing.T) {
	def := AgentDefinition{
		Tools: []string{"Glob", "Grep", "FileRead"},
	}
	if !def.IsToolAllowed("Glob") {
		t.Error("Glob should be allowed")
	}
	if !def.IsToolAllowed("glob") {
		t.Error("glob (lowercase) should be allowed (case-insensitive)")
	}
	if def.IsToolAllowed("Bash") {
		t.Error("Bash should NOT be allowed when not in list")
	}
}

func TestIsToolAllowed_EmptyMeansAll(t *testing.T) {
	def := AgentDefinition{Tools: nil}
	if !def.IsToolAllowed("Bash") {
		t.Error("empty tools list should allow all tools")
	}
	if !def.IsToolAllowed("FileRead") {
		t.Error("empty tools list should allow all tools")
	}
}

func TestIsToolAllowed_DenyList(t *testing.T) {
	def := AgentDefinition{
		Tools:           nil,
		DisallowedTools: []string{"Bash"},
	}
	if def.IsToolAllowed("Bash") {
		t.Error("Bash should be blocked by DisallowedTools")
	}
	if !def.IsToolAllowed("FileRead") {
		t.Error("FileRead should still be allowed")
	}
}

// ---------------------------------------------------------------------------
// BuildToolDescription
// ---------------------------------------------------------------------------

func TestBuildToolDescription_ContainsSections(t *testing.T) {
	agents := GetBuiltInAgents()
	desc := BuildToolDescription(agents, false)

	sections := []string{
		"Available Agent Types",
		"Usage Notes",
		"Examples",
		"When NOT to Use",
	}
	for _, s := range sections {
		if !strings.Contains(desc, s) {
			t.Errorf("description missing section %q", s)
		}
	}

	for _, a := range agents {
		if !strings.Contains(desc, a.AgentType) {
			t.Errorf("description missing agent type %q", a.AgentType)
		}
	}
}

func TestBuildToolDescription_CoordinatorMode(t *testing.T) {
	agents := GetBuiltInAgents()
	desc := BuildToolDescription(agents, true)
	if !strings.Contains(desc, "coordinator") {
		t.Error("coordinator mode description should mention coordinator")
	}
}

// ---------------------------------------------------------------------------
// InputSchema and Tool metadata
// ---------------------------------------------------------------------------

func TestAgentInputSchema_HasRequiredFields(t *testing.T) {
	pool := newTestPool()
	tool := NewTool(pool, nil, "/tmp")

	schema := tool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("schema type = %q, want %q", schema.Type, "object")
	}

	requiredProps := []string{"prompt"}
	for _, r := range requiredProps {
		found := false
		for _, req := range schema.Required {
			if req == r {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required property %q", r)
		}
	}

	expectedProps := []string{"prompt", "subagent_type", "name", "description", "model", "isolation", "run_in_background"}
	for _, p := range expectedProps {
		if _, ok := schema.Properties[p]; !ok {
			t.Errorf("missing property %q in schema", p)
		}
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello world", "hello-world"},
		{"MyAgent_Test", "myagent-test"},
		{"UPPER", "upper"},
		{"", "unnamed"},
		{"a!@#b", "ab"},
		{strings.Repeat("a", 50), strings.Repeat("a", 30)},
	}
	for _, tt := range tests {
		got := sanitizeBranchName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestPool() *coordinator.WorkerPool {
	return coordinator.NewWorkerPool()
}
