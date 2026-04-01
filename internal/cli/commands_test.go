package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// CommandRegistry basics
// ---------------------------------------------------------------------------

func TestCommandRegistry_RegisterAndFind(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{
		Name:        "test",
		Description: "a test command",
		Handler:     func(_ string, _ *CommandContext) error { return nil },
	})

	cmd := reg.Find("test")
	if cmd == nil {
		t.Fatal("Find('test') returned nil")
	}
	if cmd.Description != "a test command" {
		t.Errorf("Description = %q", cmd.Description)
	}
}

func TestCommandRegistry_FindAlias(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{
		Name:    "exit",
		Aliases: []string{"quit", "q"},
		Handler: func(_ string, _ *CommandContext) error { return nil },
	})

	if reg.Find("quit") == nil {
		t.Error("Find('quit') should resolve alias")
	}
	if reg.Find("q") == nil {
		t.Error("Find('q') should resolve alias")
	}
	if reg.Find("nonexistent") != nil {
		t.Error("Find('nonexistent') should return nil")
	}
}

func TestCommandRegistry_FindStripsSlash(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{
		Name:    "help",
		Handler: func(_ string, _ *CommandContext) error { return nil },
	})

	if reg.Find("/help") == nil {
		t.Error("Find('/help') should strip leading slash")
	}
}

func TestCommandRegistry_All(t *testing.T) {
	reg := NewCommandRegistry()
	reg.Register(&Command{Name: "beta", Handler: func(_ string, _ *CommandContext) error { return nil }})
	reg.Register(&Command{Name: "alpha", Handler: func(_ string, _ *CommandContext) error { return nil }})

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() len = %d, want 2", len(all))
	}
	if all[0].Name != "alpha" {
		t.Errorf("All()[0].Name = %q, want 'alpha' (sorted)", all[0].Name)
	}
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

func TestCommandRegistry_Execute_NotSlash(t *testing.T) {
	reg := NewCommandRegistry()
	handled, err := reg.Execute("hello", nil)
	if handled {
		t.Error("non-slash input should not be handled")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCommandRegistry_Execute_UnknownCommand(t *testing.T) {
	reg := NewCommandRegistry()
	handled, err := reg.Execute("/doesnotexist", nil)
	if !handled {
		t.Error("slash command should be handled")
	}
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestCommandRegistry_Execute_WithArgs(t *testing.T) {
	reg := NewCommandRegistry()
	var gotArgs string
	reg.Register(&Command{
		Name: "model",
		Handler: func(args string, ctx *CommandContext) error {
			gotArgs = args
			return nil
		},
	})

	handled, err := reg.Execute("/model claude-3.5", &CommandContext{})
	if !handled {
		t.Error("should be handled")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if gotArgs != "claude-3.5" {
		t.Errorf("args = %q, want 'claude-3.5'", gotArgs)
	}
}

// ---------------------------------------------------------------------------
// RegisterDefaultCommands — smoke test
// ---------------------------------------------------------------------------

func TestRegisterDefaultCommands_PopulatesRegistry(t *testing.T) {
	reg := NewCommandRegistry()
	RegisterDefaultCommands(reg)

	expectedCmds := []string{
		"help", "exit", "clear", "compact", "model", "config", "version",
		"cost", "status", "resume", "session", "export", "memory",
		"commit", "diff", "doctor",
		"permissions", "mcp", "plan", "tasks", "agents", "skills",
		"plugins", "theme", "vim", "buddy", "review",
	}

	all := reg.All()
	if len(all) < len(expectedCmds) {
		t.Errorf("expected at least %d commands, got %d", len(expectedCmds), len(all))
	}

	for _, name := range expectedCmds {
		if reg.Find(name) == nil {
			t.Errorf("missing command: %s", name)
		}
	}
}

func TestRegisterDefaultCommands_ExitAlias(t *testing.T) {
	reg := NewCommandRegistry()
	RegisterDefaultCommands(reg)

	if reg.Find("quit") == nil {
		t.Error("'quit' should be an alias for 'exit'")
	}
}

// ---------------------------------------------------------------------------
// Individual command handlers — no-panic tests with mock context
// ---------------------------------------------------------------------------

func newMockContext() *CommandContext {
	return &CommandContext{
		Model:          "claude-3.5",
		Verbose:        false,
		PermissionMode: "default",
		SessionID:      "test-session",
		TokensIn:       1000,
		TokensOut:      500,
		CostUSD:        0.05,
		CWD:            ".",
		Messages:       nil,
		MessageCount:   5,
		StartTime:      time.Now(),
	}
}

func TestCmdExit_ReturnsErrExit(t *testing.T) {
	err := cmdExit("", newMockContext())
	if !errors.Is(err, ErrExit) {
		t.Errorf("cmdExit should return ErrExit, got %v", err)
	}
}

func TestCmdClear_NoPanic(t *testing.T) {
	err := cmdClear("", newMockContext())
	if err != nil {
		t.Errorf("cmdClear: %v", err)
	}
}

func TestCmdCompact_NoPanic(t *testing.T) {
	err := cmdCompact("", newMockContext())
	if err != nil {
		t.Errorf("cmdCompact: %v", err)
	}
}

func TestCmdModel_ShowAndChange(t *testing.T) {
	ctx := newMockContext()
	if err := cmdModel("", ctx); err != nil {
		t.Errorf("cmdModel show: %v", err)
	}

	if err := cmdModel("new-model", ctx); err != nil {
		t.Errorf("cmdModel change: %v", err)
	}
	if ctx.Model != "new-model" {
		t.Errorf("Model = %q, want %q", ctx.Model, "new-model")
	}
}

func TestCmdConfig_NoPanic(t *testing.T) {
	if err := cmdConfig("", newMockContext()); err != nil {
		t.Errorf("cmdConfig: %v", err)
	}
	if err := cmdConfig("set something", newMockContext()); err != nil {
		t.Errorf("cmdConfig with args: %v", err)
	}
}

func TestCmdVersion_NoPanic(t *testing.T) {
	if err := cmdVersion("", newMockContext()); err != nil {
		t.Errorf("cmdVersion: %v", err)
	}
}

func TestCmdCost_NoPanic(t *testing.T) {
	if err := cmdCost("", newMockContext()); err != nil {
		t.Errorf("cmdCost: %v", err)
	}
}

func TestCmdStatus_NoPanic(t *testing.T) {
	if err := cmdStatus("", newMockContext()); err != nil {
		t.Errorf("cmdStatus: %v", err)
	}
}

func TestCmdPermissions_ShowAndChange(t *testing.T) {
	ctx := newMockContext()
	if err := cmdPermissions("", ctx); err != nil {
		t.Errorf("cmdPermissions show: %v", err)
	}

	if err := cmdPermissions("auto", ctx); err != nil {
		t.Errorf("cmdPermissions set: %v", err)
	}
	if ctx.PermissionMode != "auto" {
		t.Errorf("PermissionMode = %q, want %q", ctx.PermissionMode, "auto")
	}
}

func TestCmdBuddy_Toggles(t *testing.T) {
	ctx := newMockContext()
	ctx.BuddyEnabled = false

	if err := cmdBuddy("", ctx); err != nil {
		t.Errorf("cmdBuddy: %v", err)
	}
	if !ctx.BuddyEnabled {
		t.Error("buddy should be enabled after first toggle")
	}

	if err := cmdBuddy("", ctx); err != nil {
		t.Errorf("cmdBuddy: %v", err)
	}
	if ctx.BuddyEnabled {
		t.Error("buddy should be disabled after second toggle")
	}
}

func TestCmdSkills_NoPanic(t *testing.T) {
	if err := cmdSkills("", newMockContext()); err != nil {
		t.Errorf("cmdSkills: %v", err)
	}
}

func TestCmdPlugins_NoPanic(t *testing.T) {
	if err := cmdPlugins("", newMockContext()); err != nil {
		t.Errorf("cmdPlugins: %v", err)
	}
}

func TestCmdTheme_NoPanic(t *testing.T) {
	if err := cmdTheme("", newMockContext()); err != nil {
		t.Errorf("cmdTheme show: %v", err)
	}
	if err := cmdTheme("dark", newMockContext()); err != nil {
		t.Errorf("cmdTheme set: %v", err)
	}
}

func TestCmdVim_NoPanic(t *testing.T) {
	if err := cmdVim("", newMockContext()); err != nil {
		t.Errorf("cmdVim: %v", err)
	}
}

func TestCmdMCP_NilStateStore(t *testing.T) {
	ctx := newMockContext()
	ctx.StateStore = nil
	if err := cmdMCP("", ctx); err != nil {
		t.Errorf("cmdMCP with nil StateStore: %v", err)
	}
}

func TestCmdTasks_NilStateStore(t *testing.T) {
	ctx := newMockContext()
	ctx.StateStore = nil
	if err := cmdTasks("", ctx); err != nil {
		t.Errorf("cmdTasks with nil StateStore: %v", err)
	}
}

func TestCmdAgents_NilStateStore(t *testing.T) {
	ctx := newMockContext()
	ctx.StateStore = nil
	if err := cmdAgents("", ctx); err != nil {
		t.Errorf("cmdAgents with nil StateStore: %v", err)
	}
}

func TestCmdPlan_Toggle(t *testing.T) {
	ctx := newMockContext()
	ctx.PermissionMode = "default"

	if err := cmdPlan("", ctx); err != nil {
		t.Errorf("cmdPlan enable: %v", err)
	}
	if ctx.PermissionMode != "plan" {
		t.Errorf("PermissionMode = %q, want 'plan'", ctx.PermissionMode)
	}

	if err := cmdPlan("", ctx); err != nil {
		t.Errorf("cmdPlan disable: %v", err)
	}
	if ctx.PermissionMode != "default" {
		t.Errorf("PermissionMode = %q, want 'default'", ctx.PermissionMode)
	}
}

// ---------------------------------------------------------------------------
// Session commands — nil storage safety
// ---------------------------------------------------------------------------

func TestCmdResume_NilStorage(t *testing.T) {
	ctx := newMockContext()
	ctx.Storage = nil
	if err := cmdResume("", ctx); err != nil {
		t.Errorf("cmdResume with nil storage: %v", err)
	}
}

func TestCmdSession_NoActiveSession(t *testing.T) {
	ctx := newMockContext()
	ctx.SessionID = ""
	if err := cmdSession("", ctx); err != nil {
		t.Errorf("cmdSession with no session: %v", err)
	}
}

func TestCmdExport_NoMessages(t *testing.T) {
	ctx := newMockContext()
	ctx.Messages = nil
	if err := cmdExport("", ctx); err != nil {
		t.Errorf("cmdExport with no messages: %v", err)
	}
}

// ---------------------------------------------------------------------------
// parseExportArgs
// ---------------------------------------------------------------------------

func TestParseExportArgs(t *testing.T) {
	tests := []struct {
		args      string
		sessionID string
		wantFmt   string
		wantPath  string
	}{
		{"json", "abc", "json", "./session-abc.json"},
		{"markdown", "abc", "markdown", "./session-abc.md"},
		{"json /tmp/out.json", "abc", "json", "/tmp/out.json"},
		{"", "", "markdown", "./session-conversation.md"},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			format, path := parseExportArgs(tt.args, tt.sessionID)
			if format != tt.wantFmt {
				t.Errorf("format = %q, want %q", format, tt.wantFmt)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// generateCommitMessage
// ---------------------------------------------------------------------------

func TestGenerateCommitMessage(t *testing.T) {
	t.Run("single file", func(t *testing.T) {
		stat := " main.go | 5 +++--\n"
		msg := generateCommitMessage(stat)
		if msg == "" {
			t.Error("message should not be empty")
		}
	})

	t.Run("multiple files", func(t *testing.T) {
		stat := " main.go | 5 +++--\n config.go | 10 +++++++---\n"
		msg := generateCommitMessage(stat)
		if msg == "" {
			t.Error("message should not be empty")
		}
	})

	t.Run("empty", func(t *testing.T) {
		msg := generateCommitMessage("")
		if msg == "" {
			t.Error("message should not be empty")
		}
	})
}

// ---------------------------------------------------------------------------
// mcpStatusIcon / taskStatusIcon
// ---------------------------------------------------------------------------

func TestMcpStatusIcon(t *testing.T) {
	tests := []string{"connected", "connecting", "error", "unknown"}
	for _, status := range tests {
		icon := mcpStatusIcon(status)
		if icon == "" {
			t.Errorf("mcpStatusIcon(%q) returned empty", status)
		}
	}
}

func TestTaskStatusIcon(t *testing.T) {
	statuses := []types.TaskStatus{
		types.TaskRunning, types.TaskPending, types.TaskComplete,
		types.TaskFailed, types.TaskStopped, "unknown",
	}
	for _, s := range statuses {
		icon := taskStatusIcon(s)
		if icon == "" {
			t.Errorf("taskStatusIcon(%q) returned empty", s)
		}
	}
}

// ---------------------------------------------------------------------------
// New command handlers — no-panic tests
// ---------------------------------------------------------------------------

func TestCmdHelp_NoPanic(t *testing.T) {
	reg := NewCommandRegistry()
	RegisterDefaultCommands(reg)
	setDefaultRegistry(reg)
	defer setDefaultRegistry(nil)

	if err := cmdHelp("", newMockContext()); err != nil {
		t.Errorf("cmdHelp: %v", err)
	}
}

func TestCmdHelp_NilRegistry(t *testing.T) {
	prev := defaultRegistry
	setDefaultRegistry(nil)
	defer setDefaultRegistry(prev)

	if err := cmdHelp("", newMockContext()); err != nil {
		t.Errorf("cmdHelp with nil registry: %v", err)
	}
}

func TestCmdDoctor_NoPanic(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = t.TempDir()
	if err := cmdDoctor("", ctx); err != nil {
		t.Errorf("cmdDoctor: %v", err)
	}
}

func TestCmdReview_NoPanic(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = t.TempDir()
	if err := cmdReview("", ctx); err != nil {
		t.Logf("cmdReview returned error (expected if not a git repo): %v", err)
	}
}

func TestCmdMemory_NoPanic(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = t.TempDir()
	if err := cmdMemory("", ctx); err != nil {
		t.Errorf("cmdMemory show: %v", err)
	}
}

func TestCmdMemory_AppendAndShow(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = t.TempDir()

	if err := cmdMemory("remember this fact", ctx); err != nil {
		t.Fatalf("cmdMemory append: %v", err)
	}
	if err := cmdMemory("", ctx); err != nil {
		t.Errorf("cmdMemory show after append: %v", err)
	}
}

func TestCmdDiff_NoPanic(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = t.TempDir()
	if err := cmdDiff("", ctx); err != nil {
		t.Logf("cmdDiff returned error (expected if not a git repo): %v", err)
	}
}

func TestCmdDiff_CachedFlag(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = t.TempDir()
	if err := cmdDiff("--cached", ctx); err != nil {
		t.Logf("cmdDiff --cached returned error (expected if not a git repo): %v", err)
	}
}

// ---------------------------------------------------------------------------
// capitalizeFirst
// ---------------------------------------------------------------------------

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "Hello"},
		{"", ""},
		{"A", "A"},
		{"already", "Already"},
	}
	for _, tt := range tests {
		got := capitalizeFirst(tt.input)
		if got != tt.want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// colorizeDiff
// ---------------------------------------------------------------------------

func TestColorizeDiff(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 unchanged
-removed
+added
+added2`

	colored := colorizeDiff(diff)
	if colored == "" {
		t.Error("colorizeDiff returned empty")
	}
	if len(colored) <= len(diff) {
		t.Error("colored output should be longer than input due to ANSI codes")
	}
}

// ---------------------------------------------------------------------------
// Command execution — registry integration
// ---------------------------------------------------------------------------

func TestExecute_AllDefaultCommands_NoPanic(t *testing.T) {
	reg := NewCommandRegistry()
	RegisterDefaultCommands(reg)
	setDefaultRegistry(reg)
	defer setDefaultRegistry(nil)

	ctx := newMockContext()
	ctx.CWD = t.TempDir()

	commandsToTest := []string{
		"/help", "/clear", "/compact", "/model", "/config", "/version",
		"/cost", "/status", "/skills", "/plugins", "/theme", "/vim",
		"/buddy", "/permissions", "/plan",
		"/session", "/export",
	}

	for _, cmd := range commandsToTest {
		t.Run(cmd, func(t *testing.T) {
			handled, err := reg.Execute(cmd, ctx)
			if !handled {
				t.Errorf("%s should be handled", cmd)
			}
			if err != nil && !errors.Is(err, ErrExit) {
				t.Logf("%s returned error: %v (may be expected)", cmd, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// effectiveCWD
// ---------------------------------------------------------------------------

func TestEffectiveCWD_FromContext(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = "/tmp/test"
	got := effectiveCWD(ctx)
	if got != "/tmp/test" {
		t.Errorf("effectiveCWD = %q, want /tmp/test", got)
	}
}

func TestEffectiveCWD_FallsBackToGetwd(t *testing.T) {
	ctx := newMockContext()
	ctx.CWD = ""
	got := effectiveCWD(ctx)
	if got == "" {
		t.Error("effectiveCWD should not return empty")
	}
}
