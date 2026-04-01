package bash

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBashToolBasicExecution(t *testing.T) {
	tool := New()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data, ok := result.Data.(string)
	if !ok {
		t.Fatalf("result.Data is %T, want string", result.Data)
	}
	if !strings.Contains(data, "hello") {
		t.Errorf("output = %q, want to contain 'hello'", data)
	}
}

func TestBashToolExitCode(t *testing.T) {
	tool := New()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{
		"command": "exit 42",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data, ok := result.Data.(string)
	if !ok {
		t.Fatalf("result.Data is %T, want string", result.Data)
	}
	if !strings.Contains(data, "Exit code: 42") {
		t.Errorf("output = %q, want to contain 'Exit code: 42'", data)
	}
}

func TestBashToolTimeout(t *testing.T) {
	tool := New()
	ctx := context.Background()

	start := time.Now()
	result, err := tool.Call(ctx, map[string]interface{}{
		"command": "sleep 30",
		"timeout": float64(200),
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	if elapsed > 5*time.Second {
		t.Errorf("took %v, expected to time out quickly", elapsed)
	}

	data, ok := result.Data.(string)
	if !ok {
		t.Fatalf("result.Data is %T, want string", result.Data)
	}
	if !strings.Contains(data, "Exit code:") {
		t.Errorf("expected exit code in output, got %q", data)
	}
}

func TestBashToolDangerousCommand(t *testing.T) {
	tool := New()
	ctx := context.Background()

	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero of=/dev/sda",
		":(){:|:&};:",
	}

	for _, cmd := range dangerous {
		t.Run(cmd, func(t *testing.T) {
			_, err := tool.Call(ctx, map[string]interface{}{
				"command": cmd,
			})
			if err == nil {
				t.Error("expected error for dangerous command")
			}
		})
	}
}

func TestBashToolMetadata(t *testing.T) {
	tool := New()

	if tool.Name() != "Bash" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "Bash")
	}
	if tool.IsReadOnly(nil) {
		t.Error("Bash should not be read-only")
	}
	if !tool.IsEnabled() {
		t.Error("Bash should be enabled")
	}
}

func TestBashToolIsDestructive(t *testing.T) {
	tool := New()

	if tool.IsDestructive(map[string]interface{}{"command": "echo hi"}) {
		t.Error("echo should not be destructive")
	}
	if !tool.IsDestructive(map[string]interface{}{"command": "rm -rf /"}) {
		t.Error("rm -rf / should be destructive")
	}
}
