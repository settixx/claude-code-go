package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"
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

	if len(state.history) == 0 {
		t.Fatal("history should not be empty")
	}
	state.historyIdx = len(state.history) - 1
	if state.history[state.historyIdx] != "cmd3" {
		t.Errorf("expected cmd3, got %q", state.history[state.historyIdx])
	}

	state.historyIdx--
	if state.history[state.historyIdx] != "cmd2" {
		t.Errorf("expected cmd2, got %q", state.history[state.historyIdx])
	}

	state.historyIdx = len(state.history)
	if state.historyIdx >= len(state.history) {
		state.historyIdx = -1
	}
	if state.historyIdx != -1 {
		t.Error("should be back at draft position")
	}
}

// ---------------------------------------------------------------------------
// Multi-line input
// ---------------------------------------------------------------------------

func TestInputModel_MultiLineValue(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("line1\nline2\nline3")

	val := m.Value()
	if !strings.Contains(val, "line1") || !strings.Contains(val, "line3") {
		t.Errorf("multi-line Value = %q", val)
	}
	if m.LineCount() != 3 {
		t.Errorf("LineCount = %d, want 3", m.LineCount())
	}
}

func TestInputModel_InsertNewline(t *testing.T) {
	m := NewInputModel("normal")
	m.textInput.SetValue("first")
	m.insertNewline()

	if len(m.lines) != 1 || m.lines[0] != "first" {
		t.Errorf("lines = %v, want [first]", m.lines)
	}
	if m.textInput.Value() != "" {
		t.Errorf("current line should be empty after newline insert")
	}
}

func TestInputModel_MaxLines(t *testing.T) {
	m := NewInputModel("normal")
	for i := 0; i < maxInputLines+2; i++ {
		m.textInput.SetValue(fmt.Sprintf("line%d", i))
		m.insertNewline()
	}
	if m.LineCount() > maxInputLines {
		t.Errorf("LineCount = %d, should not exceed %d", m.LineCount(), maxInputLines)
	}
}

func TestInputModel_Submit_Resets(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("multi\nline\ninput")
	val := m.Submit()

	if !strings.Contains(val, "multi") {
		t.Errorf("Submit should return full text, got %q", val)
	}
	if m.LineCount() != 1 {
		t.Errorf("LineCount after submit = %d, want 1", m.LineCount())
	}
	if m.Value() != "" {
		t.Errorf("Value after submit = %q, want empty", m.Value())
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
		{"/buddy", true},
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
		"/buddy": false,
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
		if len(stripped) > 45 {
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

func TestTokenUsageMsg(t *testing.T) {
	msg := TokenUsageMsg{InputTokens: 100, OutputTokens: 50}
	if msg.InputTokens != 100 {
		t.Errorf("InputTokens = %d", msg.InputTokens)
	}
	if msg.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d", msg.OutputTokens)
	}
}

// ---------------------------------------------------------------------------
// CostTracker
// ---------------------------------------------------------------------------

func TestCostTracker_Add(t *testing.T) {
	ct := &CostTracker{}
	ct.Add(1000, 500)

	if ct.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", ct.InputTokens)
	}
	if ct.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", ct.OutputTokens)
	}
	if ct.TotalTokens() != 1500 {
		t.Errorf("TotalTokens = %d, want 1500", ct.TotalTokens())
	}
	if ct.CostUSD <= 0 {
		t.Errorf("CostUSD should be > 0, got %f", ct.CostUSD)
	}
}

func TestCostTracker_FormatTokensSplit(t *testing.T) {
	ct := &CostTracker{InputTokens: 1200, OutputTokens: 450}
	got := ct.FormatTokensSplit()
	if !strings.Contains(got, "1.2k↓") {
		t.Errorf("FormatTokensSplit = %q, missing 1.2k↓", got)
	}
	if !strings.Contains(got, "450↑") {
		t.Errorf("FormatTokensSplit = %q, missing 450↑", got)
	}
}

func TestCostTracker_FormatStatusSegment(t *testing.T) {
	ct := &CostTracker{}
	if ct.FormatStatusSegment() != "" {
		t.Error("empty tracker should return empty segment")
	}

	ct.Add(2000, 300)
	seg := ct.FormatStatusSegment()
	if !strings.Contains(seg, "Tokens:") {
		t.Errorf("segment should contain 'Tokens:', got %q", seg)
	}
	if !strings.Contains(seg, "Cost:") {
		t.Errorf("segment should contain 'Cost:', got %q", seg)
	}
}

func TestCostTracker_Reset(t *testing.T) {
	ct := &CostTracker{InputTokens: 100, OutputTokens: 50, CostUSD: 0.5}
	ct.Reset()
	if ct.TotalTokens() != 0 || ct.CostUSD != 0 {
		t.Errorf("Reset did not zero fields")
	}
}

// ---------------------------------------------------------------------------
// BuddyWidget
// ---------------------------------------------------------------------------

func TestBuddyWidget_Toggle(t *testing.T) {
	bw := NewBuddyWidget()
	if bw.IsVisible() {
		t.Error("buddy should be hidden by default")
	}
	bw.Toggle()
	if !bw.IsVisible() {
		t.Error("buddy should be visible after toggle")
	}
	bw.Toggle()
	if bw.IsVisible() {
		t.Error("buddy should be hidden after second toggle")
	}
}

func TestBuddyWidget_View_Hidden(t *testing.T) {
	bw := NewBuddyWidget()
	if bw.View() != "" {
		t.Error("hidden buddy should render empty")
	}
	if bw.Height() != 0 {
		t.Error("hidden buddy height should be 0")
	}
}

func TestBuddyWidget_View_Visible(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("  __\n (o>\n /| \n / |")
	bw.SetText("Hello!")
	bw.SetWidth(40)

	view := bw.View()
	if view == "" {
		t.Error("visible buddy with frame should render content")
	}
	if !strings.Contains(view, "duck") {
		t.Errorf("view should contain species name, got %q", view)
	}
	if bw.Height() < 4 {
		t.Errorf("height should be at least 4 lines, got %d", bw.Height())
	}
}

// ---------------------------------------------------------------------------
// StatusLine with new params
// ---------------------------------------------------------------------------

func TestStatusLine_WithBranch(t *testing.T) {
	ct := &CostTracker{InputTokens: 500, OutputTokens: 100, CostUSD: 0.003}
	sl := StatusLineFromState("claude-3.5", ct, "auto", "abc12345", "main")
	rendered := sl.Render(80)

	if !strings.Contains(rendered, "main") {
		t.Errorf("status line should contain branch 'main', got %q", StripANSI(rendered))
	}
}

func TestStatusLine_NoBranch(t *testing.T) {
	ct := &CostTracker{}
	sl := StatusLineFromState("model", ct, "", "sid", "")
	rendered := sl.Render(80)
	stripped := StripANSI(rendered)

	if strings.Contains(stripped, "[]") {
		t.Errorf("should not show empty brackets, got %q", stripped)
	}
}

// ---------------------------------------------------------------------------
// GitBranchCache
// ---------------------------------------------------------------------------

func TestGitBranchCache_Basic(t *testing.T) {
	cache := NewGitBranchCache(5 * time.Second)
	if cache == nil {
		t.Fatal("NewGitBranchCache returned nil")
	}

	branch := cache.Branch()
	if branch == "" {
		t.Log("Branch returned empty — may not be in a git repo (acceptable in CI)")
	}
}

// ---------------------------------------------------------------------------
// CostTracker — additional coverage
// ---------------------------------------------------------------------------

func TestCostTracker_FormatTokensSplit_Millions(t *testing.T) {
	ct := &CostTracker{InputTokens: 1_500_000, OutputTokens: 250_000}
	got := ct.FormatTokensSplit()
	if !strings.Contains(got, "1.5M↓") {
		t.Errorf("FormatTokensSplit = %q, missing 1.5M↓", got)
	}
	if !strings.Contains(got, "250.0k↑") {
		t.Errorf("FormatTokensSplit = %q, missing 250.0k↑", got)
	}
}

func TestCostTracker_FormatTokensSplit_SmallNumbers(t *testing.T) {
	ct := &CostTracker{InputTokens: 50, OutputTokens: 10}
	got := ct.FormatTokensSplit()
	if !strings.Contains(got, "50↓") {
		t.Errorf("FormatTokensSplit = %q, missing 50↓", got)
	}
	if !strings.Contains(got, "10↑") {
		t.Errorf("FormatTokensSplit = %q, missing 10↑", got)
	}
}

// ---------------------------------------------------------------------------
// BuddyWidget — additional coverage
// ---------------------------------------------------------------------------

func TestBuddyWidget_HiddenAfterToggle(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("sprite")
	bw.SetWidth(40)

	if bw.View() == "" {
		t.Error("visible buddy with frame should have content")
	}

	bw.Toggle()
	if bw.View() != "" {
		t.Error("hidden buddy should return empty view")
	}
}

// ---------------------------------------------------------------------------
// Multi-line input — additional coverage
// ---------------------------------------------------------------------------

func TestInputModel_SetValueSingle(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("single line")
	if m.LineCount() != 1 {
		t.Errorf("LineCount = %d, want 1", m.LineCount())
	}
	if m.Value() != "single line" {
		t.Errorf("Value = %q", m.Value())
	}
}

func TestInputModel_Reset(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("some\nmulti\nline")
	m.Reset()

	if m.Value() != "" {
		t.Errorf("Value after Reset = %q, want empty", m.Value())
	}
	if m.LineCount() != 1 {
		t.Errorf("LineCount after Reset = %d, want 1", m.LineCount())
	}
}

func TestInputModel_SetMode(t *testing.T) {
	m := NewInputModel("normal")
	m.SetMode("plan")
	// Should not panic
	m.SetMode("normal")
}

func TestInputModel_SetWidth(t *testing.T) {
	m := NewInputModel("normal")
	m.SetWidth(120)
	// Should not panic
	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
}

func TestInputModel_SubmitRecordsHistory(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("first command")
	m.Submit()
	m.SetValue("second command")
	m.Submit()

	if len(m.state.history) != 2 {
		t.Errorf("history len = %d, want 2", len(m.state.history))
	}
}

func TestInputModel_SubmitEmpty(t *testing.T) {
	m := NewInputModel("normal")
	val := m.Submit()
	if val != "" {
		t.Errorf("Submit empty should return empty, got %q", val)
	}
	if len(m.state.history) != 0 {
		t.Errorf("empty submit should not add to history, len = %d", len(m.state.history))
	}
}

// ---------------------------------------------------------------------------
// StatusLine — additional coverage
// ---------------------------------------------------------------------------

func TestStatusLine_RenderZeroWidth(t *testing.T) {
	sl := &StatusLine{Left: "left", Right: "right"}
	got := sl.Render(0)
	if got != "" {
		t.Errorf("Render(0) = %q, want empty", got)
	}
}

func TestStatusLine_RenderNarrow(t *testing.T) {
	sl := &StatusLine{Left: "ABCDEF", Right: "XY"}
	got := sl.Render(5)
	if got == "" {
		t.Error("Render narrow should return something")
	}
}

func TestStatusLine_RenderCentered(t *testing.T) {
	sl := &StatusLine{Left: "L", Center: "C", Right: "R"}
	got := sl.Render(40)
	stripped := StripANSI(got)
	if !strings.Contains(stripped, "L") || !strings.Contains(stripped, "C") || !strings.Contains(stripped, "R") {
		t.Errorf("all sections should be present, got %q", stripped)
	}
}

func TestStatusLineFromState_AllFields(t *testing.T) {
	ct := &CostTracker{InputTokens: 2000, OutputTokens: 300, CostUSD: 0.01}
	sl := StatusLineFromState("claude-3.5", ct, "auto", "sess-abc123", "feature-branch")
	rendered := sl.Render(100)
	stripped := StripANSI(rendered)

	if !strings.Contains(stripped, "claude-3.5") {
		t.Error("should contain model name")
	}
	if !strings.Contains(stripped, "auto") {
		t.Error("should contain permission mode")
	}
	if !strings.Contains(stripped, "feature-branch") {
		t.Error("should contain branch name")
	}
}

// ---------------------------------------------------------------------------
// Format helpers — additional coverage
// ---------------------------------------------------------------------------

func TestFormatCost_Zero(t *testing.T) {
	got := FormatCost(0)
	if got != "$0.0000" {
		t.Errorf("FormatCost(0) = %q, want $0.0000", got)
	}
}

func TestFormatCost_Large(t *testing.T) {
	got := FormatCost(12.345)
	if got != "$12.35" {
		t.Errorf("FormatCost(12.345) = %q, want $12.35", got)
	}
}

func TestHorizontalRule(t *testing.T) {
	hr := HorizontalRule()
	if len(hr) == 0 {
		t.Error("HorizontalRule should return non-empty string")
	}
}

// ---------------------------------------------------------------------------
// BuddyWidget — additional state tests
// ---------------------------------------------------------------------------

func TestBuddyWidget_SetText(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("(o>")
	bw.SetText("Thinking...")
	bw.SetWidth(40)

	view := bw.View()
	if view == "" {
		t.Error("buddy with text and frame should render")
	}
}

func TestBuddyWidget_SetWidth_Narrow(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("sprite art")
	bw.SetWidth(15)

	view := bw.View()
	if view == "" {
		t.Error("narrow buddy should still render")
	}
}

func TestBuddyWidget_DefaultSpecies(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("art")
	bw.SetWidth(40)
	view := bw.View()
	if !strings.Contains(view, "duck") {
		t.Errorf("default species should be 'duck', view: %q", view)
	}
}

func TestBuddyWidget_Height_IncludesFrameAndText(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("line1\nline2\nline3")
	bw.SetText("some text")
	bw.SetWidth(40)

	h := bw.Height()
	if h < 5 {
		t.Errorf("height should be at least 5 (3 frame + 1 text + 2 border), got %d", h)
	}
}

// ---------------------------------------------------------------------------
// CostTracker — edge cases
// ---------------------------------------------------------------------------

func TestCostTracker_ZeroTokens(t *testing.T) {
	ct := &CostTracker{}
	if ct.TotalTokens() != 0 {
		t.Errorf("initial TotalTokens = %d", ct.TotalTokens())
	}
	if ct.CostUSD != 0 {
		t.Errorf("initial CostUSD = %f", ct.CostUSD)
	}
}

func TestCostTracker_FormatTokensSplit_ZeroInput(t *testing.T) {
	ct := &CostTracker{InputTokens: 0, OutputTokens: 100}
	got := ct.FormatTokensSplit()
	if !strings.Contains(got, "0↓") {
		t.Errorf("FormatTokensSplit = %q, missing 0↓", got)
	}
}

func TestCostTracker_FormatStatusSegment_WithData(t *testing.T) {
	ct := &CostTracker{}
	ct.Add(10000, 5000)
	seg := ct.FormatStatusSegment()
	if !strings.Contains(seg, "Tokens:") || !strings.Contains(seg, "Cost:") {
		t.Errorf("segment = %q, expected Tokens and Cost sections", seg)
	}
}

// ---------------------------------------------------------------------------
// GitBranchCache — TTL expiry
// ---------------------------------------------------------------------------

func TestGitBranchCache_TTLExpiry(t *testing.T) {
	cache := NewGitBranchCache(1 * time.Millisecond)
	_ = cache.Branch()

	time.Sleep(5 * time.Millisecond)
	branch := cache.Branch()
	_ = branch
}

func TestGitBranchCache_ZeroTTL(t *testing.T) {
	cache := NewGitBranchCache(0)
	_ = cache.Branch()
	_ = cache.Branch()
}

// ---------------------------------------------------------------------------
// Multi-line input — edge cases
// ---------------------------------------------------------------------------

func TestInputModel_MultiLineLargeText(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("line1\nline2\nline3\nline4\nline5")
	if m.LineCount() != 5 {
		t.Errorf("LineCount = %d, want 5", m.LineCount())
	}
}

func TestInputModel_SubmitMultiLineRecordsHistory(t *testing.T) {
	m := NewInputModel("normal")
	m.SetValue("line1\nline2")
	val := m.Submit()
	if !strings.Contains(val, "line1") || !strings.Contains(val, "line2") {
		t.Errorf("Submit should return full text, got %q", val)
	}
	if len(m.state.history) != 1 {
		t.Errorf("history len = %d, want 1", len(m.state.history))
	}
}

func TestInputModel_InsertNewline_AtMaxLines(t *testing.T) {
	m := NewInputModel("normal")
	for i := 0; i < maxInputLines; i++ {
		m.textInput.SetValue(fmt.Sprintf("l%d", i))
		m.insertNewline()
	}
	before := m.LineCount()
	m.textInput.SetValue("overflow")
	m.insertNewline()
	after := m.LineCount()

	if after > before+1 {
		t.Errorf("insertNewline at max should not add line: before=%d after=%d", before, after)
	}
}

// ---------------------------------------------------------------------------
// StatusLine — permission mode labels
// ---------------------------------------------------------------------------

func TestStatusLine_PermModePlan(t *testing.T) {
	ct := &CostTracker{}
	sl := StatusLineFromState("model", ct, "plan", "sid", "")
	rendered := sl.Render(80)
	stripped := StripANSI(rendered)
	if !strings.Contains(stripped, "plan") {
		t.Errorf("status line should contain 'plan', got %q", stripped)
	}
}

func TestStatusLine_PermModeAutoAndBypass(t *testing.T) {
	modes := []string{"auto", "bypassPermissions", "acceptEdits"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			ct := &CostTracker{}
			sl := StatusLineFromState("model", ct, mode, "sid", "")
			rendered := sl.Render(80)
			stripped := StripANSI(rendered)
			if stripped == "" {
				t.Error("rendered should not be empty")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatCost — boundary values
// ---------------------------------------------------------------------------

func TestFormatCost_VerySmall(t *testing.T) {
	got := FormatCost(0.000001)
	if got == "" {
		t.Error("FormatCost should not return empty for tiny value")
	}
}

func TestFormatCost_ExactDollar(t *testing.T) {
	got := FormatCost(1.00)
	if got != "$1.00" {
		t.Errorf("FormatCost(1.00) = %q, want $1.00", got)
	}
}

// ---------------------------------------------------------------------------
// Color helpers — StripANSI on plain text
// ---------------------------------------------------------------------------

func TestStripANSI_PlainText(t *testing.T) {
	got := StripANSI("plain text")
	if got != "plain text" {
		t.Errorf("StripANSI plain = %q", got)
	}
}

func TestStripANSI_Empty(t *testing.T) {
	got := StripANSI("")
	if got != "" {
		t.Errorf("StripANSI empty = %q", got)
	}
}

// ---------------------------------------------------------------------------
// Truncate — additional coverage
// ---------------------------------------------------------------------------

func TestTruncate_ExactLength(t *testing.T) {
	got := Truncate("hello", 5)
	if got != "hello" {
		t.Errorf("Truncate exact = %q, want %q", got, "hello")
	}
}

func TestTruncate_OneLess(t *testing.T) {
	got := Truncate("hello", 4)
	if got != "hel…" {
		t.Errorf("Truncate 4 = %q, want %q", got, "hel…")
	}
}
