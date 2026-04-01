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

// ---------------------------------------------------------------------------
// Security chain unit tests
// ---------------------------------------------------------------------------

func TestValidateEmpty(t *testing.T) {
	r := ValidateCommand("")
	if r.Behavior != SecurityAllow {
		t.Errorf("empty command should be allowed, got %s", r.Behavior)
	}
	r = ValidateCommand("   ")
	if r.Behavior != SecurityAllow {
		t.Errorf("whitespace-only command should be allowed, got %s", r.Behavior)
	}
}

func TestValidateIncompleteCommands(t *testing.T) {
	for _, cmd := range []string{"&& echo hi", "|| true", "; ls", ">> file"} {
		r := ValidateCommand(cmd)
		if r.Behavior != SecurityDeny {
			t.Errorf("incomplete command %q should be denied, got %s", cmd, r.Behavior)
		}
	}
}

func TestValidateCommandSubstitution(t *testing.T) {
	for _, cmd := range []string{
		"echo $(whoami)",
		"echo `hostname`",
		"echo ${HOME}",
	} {
		r := ValidateCommand(cmd)
		if r.Behavior != SecurityDeny {
			t.Errorf("command substitution %q should be denied, got %s", cmd, r.Behavior)
		}
	}
}

func TestValidateZshDangerous(t *testing.T) {
	r := ValidateCommand("zmodload zsh/net/tcp")
	if r.Behavior != SecurityDeny {
		t.Errorf("zmodload should be denied, got %s", r.Behavior)
	}
}

func TestValidateNewlines(t *testing.T) {
	r := ValidateCommand("echo foo\nrm -rf /")
	if r.Behavior != SecurityDeny {
		t.Errorf("newline command should be denied, got %s", r.Behavior)
	}
}

func TestValidateDangerousVariables(t *testing.T) {
	r := ValidateCommand("BASH_ENV=/tmp/evil.sh bash")
	if r.Behavior != SecurityDeny {
		t.Errorf("BASH_ENV set should be denied, got %s", r.Behavior)
	}
}

func TestValidateControlCharacters(t *testing.T) {
	r := ValidateCommand("echo \x01hello")
	if r.Behavior != SecurityDeny {
		t.Errorf("control char should be denied, got %s", r.Behavior)
	}
}

// ---------------------------------------------------------------------------
// Read-only validation tests
// ---------------------------------------------------------------------------

func TestReadOnlyAllowsSafeCmds(t *testing.T) {
	safe := []string{
		"ls -la",
		"cat foo.txt",
		"git status",
		"git log --oneline",
		"git diff HEAD",
		"grep -r pattern .",
		"find . -name '*.go'",
		"wc -l file.txt",
	}
	for _, cmd := range safe {
		r := ValidateReadOnly(cmd)
		if r.Behavior != SecurityAllow {
			t.Errorf("read-only should allow %q, got %s: %s", cmd, r.Behavior, r.Message)
		}
	}
}

func TestReadOnlyBlocksWriteCmds(t *testing.T) {
	blocked := []string{
		"git push origin main",
		"git commit -m msg",
		"docker run ubuntu",
		"npm install",
		"rm file.txt",
	}
	for _, cmd := range blocked {
		r := ValidateReadOnly(cmd)
		if r.Behavior != SecurityDeny {
			t.Errorf("read-only should deny %q, got %s: %s", cmd, r.Behavior, r.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// Semantics tests
// ---------------------------------------------------------------------------

func TestGrepSemantics(t *testing.T) {
	sem := GetCommandSemantic("grep pattern file")
	if sem == nil {
		t.Fatal("expected semantic for grep")
	}

	isErr, _ := sem(0, "match", "")
	if isErr {
		t.Error("grep exit 0 should not be error")
	}

	isErr, msg := sem(1, "", "")
	if isErr {
		t.Error("grep exit 1 should not be error")
	}
	if !strings.Contains(msg, "no match") {
		t.Errorf("grep exit 1 message should mention no match, got %q", msg)
	}

	isErr, _ = sem(2, "", "bad regex")
	if !isErr {
		t.Error("grep exit 2 should be error")
	}
}

func TestDiffSemantics(t *testing.T) {
	sem := GetCommandSemantic("diff a b")
	if sem == nil {
		t.Fatal("expected semantic for diff")
	}

	isErr, _ := sem(1, "< line", "")
	if isErr {
		t.Error("diff exit 1 should not be error")
	}
}

// ---------------------------------------------------------------------------
// Quoting tests
// ---------------------------------------------------------------------------

func TestExtractQuotedContent(t *testing.T) {
	q := ExtractQuotedContent(`echo "hello world" 'safe' end`)
	if !strings.Contains(q.FullyUnquoted, "echo") {
		t.Error("fully unquoted should contain echo")
	}
	if strings.Contains(q.FullyUnquoted, "hello") {
		t.Error("fully unquoted should not contain quoted 'hello'")
	}
	if strings.Contains(q.FullyUnquoted, "safe") {
		t.Error("fully unquoted should not contain single-quoted 'safe'")
	}
	if !strings.Contains(q.FullyUnquoted, "end") {
		t.Error("fully unquoted should contain 'end'")
	}
}

func TestStripSafeRedirections(t *testing.T) {
	input := "cmd 2>&1 >/dev/null"
	got := StripSafeRedirections(input)
	if strings.Contains(got, "2>&1") || strings.Contains(got, "/dev/null") {
		t.Errorf("safe redirections not stripped: %q", got)
	}
}

func TestExtractBaseCommand(t *testing.T) {
	tests := map[string]string{
		"ls -la":                  "ls",
		"git status | grep foo":  "git",
		"sudo rm -rf /":          "rm",
		"env FOO=bar python":     "FOO=bar",
	}
	for input, want := range tests {
		got := ExtractBaseCommand(input)
		if got != want {
			t.Errorf("ExtractBaseCommand(%q) = %q, want %q", input, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Read-only mode integration
// ---------------------------------------------------------------------------

func TestReadOnlyToolBlocksWrites(t *testing.T) {
	tool := NewReadOnly()
	ctx := context.Background()

	_, err := tool.Call(ctx, map[string]interface{}{
		"command": "git push origin main",
	})
	if err == nil {
		t.Error("read-only tool should block git push")
	}
}

func TestReadOnlyToolAllowsReads(t *testing.T) {
	tool := NewReadOnly()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{
		"command": "echo safe",
	})
	if err != nil {
		t.Fatalf("read-only tool should allow echo: %v", err)
	}
	data, _ := result.Data.(string)
	if !strings.Contains(data, "safe") {
		t.Errorf("expected 'safe' in output, got %q", data)
	}
}
