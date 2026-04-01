package tui

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DisplayMessage
// ---------------------------------------------------------------------------

func TestDisplayMessage_Fields(t *testing.T) {
	dm := DisplayMessage{
		Role:     "assistant",
		Content:  "Hello!",
		ToolName: "",
		Tokens:   50,
		Cost:     0.001,
	}
	if dm.Role != "assistant" {
		t.Errorf("Role = %q", dm.Role)
	}
	if dm.Tokens != 50 {
		t.Errorf("Tokens = %d", dm.Tokens)
	}
}

func TestDisplayMessage_ToolRole(t *testing.T) {
	dm := DisplayMessage{
		Role:     "tool",
		Content:  "output of bash",
		ToolName: "Bash",
	}
	if dm.ToolName != "Bash" {
		t.Errorf("ToolName = %q", dm.ToolName)
	}
}

// ---------------------------------------------------------------------------
// InputModel (unit-testable without a terminal)
// ---------------------------------------------------------------------------

func TestInputState_HistoryNavigation(t *testing.T) {
	state := InputState{
		history:    []string{"cmd1", "cmd2", "cmd3"},
		historyIdx: -1,
	}

	// Simulate historyPrev: should go to last entry
	if len(state.history) == 0 {
		t.Fatal("history should not be empty")
	}
	state.historyIdx = len(state.history) - 1
	if state.history[state.historyIdx] != "cmd3" {
		t.Errorf("expected cmd3, got %q", state.history[state.historyIdx])
	}

	// Move backwards
	state.historyIdx--
	if state.history[state.historyIdx] != "cmd2" {
		t.Errorf("expected cmd2, got %q", state.history[state.historyIdx])
	}

	// Move forward past end restores draft
	state.historyIdx = len(state.history)
	if state.historyIdx >= len(state.history) {
		state.historyIdx = -1 // back to draft
	}
	if state.historyIdx != -1 {
		t.Error("should be back at draft position")
	}
}

// ---------------------------------------------------------------------------
// Slash command detection
// ---------------------------------------------------------------------------

func TestSlashCommandDetection(t *testing.T) {
	tests := []struct {
		input   string
		isSlash bool
	}{
		{"/help", true},
		{"/exit", true},
		{"/quit", true},
		{"/clear", true},
		{"/model", true},
		{"/cost", true},
		{"/compact", true},
		{"/config", true},
		{"/resume", true},
		{"/status", true},
		{"hello", false},
		{"", false},
		{"/ ", true},
	}
	for _, tt := range tests {
		isSlash := strings.HasPrefix(tt.input, "/")
		if isSlash != tt.isSlash {
			t.Errorf("input %q: HasPrefix('/') = %v, want %v", tt.input, isSlash, tt.isSlash)
		}
	}
}

func TestSlashCommandsList(t *testing.T) {
	if len(slashCommands) < 8 {
		t.Errorf("expected at least 8 slash commands, got %d", len(slashCommands))
	}

	expected := map[string]bool{
		"/help": false, "/exit": false, "/quit": false, "/clear": false,
		"/model": false, "/cost": false, "/compact": false, "/config": false,
	}
	for _, cmd := range slashCommands {
		if _, ok := expected[cmd]; ok {
			expected[cmd] = true
		}
	}
	for cmd, found := range expected {
		if !found {
			t.Errorf("missing slash command %q", cmd)
		}
	}
}

// ---------------------------------------------------------------------------
// Color helpers
// ---------------------------------------------------------------------------

func TestStripANSI(t *testing.T) {
	colored := Red("hello")
	stripped := StripANSI(colored)
	if stripped != "hello" {
		t.Errorf("StripANSI = %q, want %q", stripped, "hello")
	}
}

func TestColorFunctions(t *testing.T) {
	funcs := map[string]func(string) string{
		"Red": Red, "Green": Green, "Yellow": Yellow,
		"Blue": Blue, "Cyan": Cyan, "Magenta": Magenta,
		"Dim": Dim, "Bold": Bold,
	}
	for name, fn := range funcs {
		result := fn("test")
		if !strings.Contains(result, "test") {
			t.Errorf("%s should contain original text", name)
		}
		if !strings.Contains(result, AnsiReset) {
			t.Errorf("%s should contain reset code", name)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell…"},
		{"", 5, ""},
		{"hi", 0, ""},
	}
	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestWrap(t *testing.T) {
	long := strings.Repeat("word ", 20)
	wrapped := Wrap(long, 40)
	for _, line := range strings.Split(wrapped, "\n") {
		stripped := StripANSI(line)
		if len(stripped) > 45 { // some tolerance for word boundaries
			t.Errorf("line too long (%d chars): %q", len(stripped), stripped)
		}
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.001, "$0.0010"},
		{0.05, "$0.05"},
		{1.23, "$1.23"},
	}
	for _, tt := range tests {
		got := FormatCost(tt.input)
		if got != tt.want {
			t.Errorf("FormatCost(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestColorIndex(t *testing.T) {
	for i := 0; i < 12; i++ {
		fn := ColorIndex(i)
		result := fn("test")
		if !strings.Contains(result, "test") {
			t.Errorf("ColorIndex(%d) output missing text", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

func TestStreamChunkMsg(t *testing.T) {
	msg := StreamChunkMsg{Text: "partial"}
	if msg.Text != "partial" {
		t.Errorf("Text = %q", msg.Text)
	}
}

func TestToolCallMsg(t *testing.T) {
	msg := ToolCallMsg{Name: "Bash", Input: `{"command":"ls"}`}
	if msg.Name != "Bash" {
		t.Errorf("Name = %q", msg.Name)
	}
}

func TestToolResultMsg(t *testing.T) {
	msg := ToolResultMsg{Name: "Bash", Result: "file1\nfile2"}
	if msg.Name != "Bash" {
		t.Errorf("Name = %q", msg.Name)
	}
}

func TestErrorMsg(t *testing.T) {
	msg := ErrorMsg{Err: fmt.Errorf("something went wrong")}
	if msg.Err == nil {
		t.Error("Err should not be nil")
	}
}

func TestPermissionRequestMsg(t *testing.T) {
	ch := make(chan bool, 1)
	msg := PermissionRequestMsg{Tool: "Bash", Input: "rm -rf /", ResponseCh: ch}
	if msg.Tool != "Bash" {
		t.Errorf("Tool = %q", msg.Tool)
	}
	ch <- true
	if !<-msg.ResponseCh {
		t.Error("expected true from channel")
	}
}
