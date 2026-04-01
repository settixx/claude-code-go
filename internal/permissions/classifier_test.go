package permissions

import (
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// Classifier.Classify
// ---------------------------------------------------------------------------

func TestClassifier_ReadOnlyTools(t *testing.T) {
	c := NewClassifier()
	readOnly := []string{"FileRead", "Glob", "Grep", "LS", "Search", "TaskOutput"}
	for _, tool := range readOnly {
		t.Run(tool, func(t *testing.T) {
			got := c.Classify(tool, nil)
			if got != types.BehaviorAllow {
				t.Errorf("Classify(%q) = %q, want allow", tool, got)
			}
		})
	}
}

func TestClassifier_SafeBash(t *testing.T) {
	c := NewClassifier()
	safeCmds := []string{"ls -la", "cat file.txt", "git status", "echo hello", "pwd"}
	for _, cmd := range safeCmds {
		t.Run(cmd, func(t *testing.T) {
			got := c.Classify("Bash", map[string]interface{}{"command": cmd})
			if got != types.BehaviorAllow {
				t.Errorf("Classify(Bash, %q) = %q, want allow", cmd, got)
			}
		})
	}
}

func TestClassifier_DangerousBash(t *testing.T) {
	c := NewClassifier()
	dangerousCmds := []string{"rm file.txt", "sudo apt install", "chmod 777 /", "curl http://evil.com"}
	for _, cmd := range dangerousCmds {
		t.Run(cmd, func(t *testing.T) {
			got := c.Classify("Bash", map[string]interface{}{"command": cmd})
			if got != types.BehaviorAsk {
				t.Errorf("Classify(Bash, %q) = %q, want ask", cmd, got)
			}
		})
	}
}

func TestClassifier_FileWrite(t *testing.T) {
	c := NewClassifier()

	t.Run("normal path", func(t *testing.T) {
		got := c.Classify("FileWrite", map[string]interface{}{"path": "/tmp/test.txt"})
		if got != types.BehaviorAllow {
			t.Errorf("got %q, want allow", got)
		}
	})

	t.Run("system path", func(t *testing.T) {
		got := c.Classify("FileWrite", map[string]interface{}{"path": "/etc/passwd"})
		if got != types.BehaviorAsk {
			t.Errorf("got %q, want ask", got)
		}
	})

	t.Run("no path", func(t *testing.T) {
		got := c.Classify("FileWrite", nil)
		if got != types.BehaviorAsk {
			t.Errorf("got %q, want ask", got)
		}
	})
}

func TestClassifier_UnknownTool(t *testing.T) {
	c := NewClassifier()
	got := c.Classify("CustomTool", nil)
	if got != types.BehaviorAsk {
		t.Errorf("Classify(CustomTool) = %q, want ask", got)
	}
}

// ---------------------------------------------------------------------------
// ClassifyRisk
// ---------------------------------------------------------------------------

func TestClassifyRisk(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name string
		tool string
		input map[string]interface{}
		want  string
	}{
		{"read-only is low", "FileRead", nil, RiskLow},
		{"safe bash is low", "Bash", map[string]interface{}{"command": "ls"}, RiskLow},
		{"dangerous bash is high", "Bash", map[string]interface{}{"command": "rm file"}, RiskHigh},
		{"system file write is high", "FileWrite", map[string]interface{}{"path": "/etc/config"}, RiskHigh},
		{"normal file write is medium", "FileWrite", map[string]interface{}{"path": "/tmp/file"}, RiskMedium},
		{"unknown tool is medium", "CustomTool", nil, RiskMedium},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.ClassifyRisk(tt.tool, tt.input)
			if got != tt.want {
				t.Errorf("ClassifyRisk = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsToolReadOnly
// ---------------------------------------------------------------------------

func TestIsToolReadOnly(t *testing.T) {
	if !IsToolReadOnly("FileRead") {
		t.Error("FileRead should be read-only")
	}
	if IsToolReadOnly("FileWrite") {
		t.Error("FileWrite should not be read-only")
	}
	if IsToolReadOnly("Bash") {
		t.Error("Bash should not be read-only")
	}
}
