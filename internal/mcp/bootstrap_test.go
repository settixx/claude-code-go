package mcp

import (
	"context"
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// Bootstrap — empty servers
// ---------------------------------------------------------------------------

func TestBootstrap_EmptyServers(t *testing.T) {
	ctx := context.Background()
	result, err := Bootstrap(ctx, nil)
	if err != nil {
		t.Fatalf("Bootstrap(nil): %v", err)
	}
	if result.Manager == nil {
		t.Fatal("Manager should not be nil")
	}
	if len(result.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result.Tools))
	}
	if len(result.Instructions) != 0 {
		t.Errorf("expected 0 instructions, got %d", len(result.Instructions))
	}
}

func TestBootstrap_EmptyMap(t *testing.T) {
	ctx := context.Background()
	result, err := Bootstrap(ctx, map[string]types.McpServerConfig{})
	if err != nil {
		t.Fatalf("Bootstrap(empty): %v", err)
	}
	if result.Manager == nil {
		t.Fatal("Manager should not be nil")
	}
}

// Bootstrap with a fake server that will fail to connect — errors should not
// prevent the result from being returned.
func TestBootstrap_FailingServer(t *testing.T) {
	ctx := context.Background()
	servers := map[string]types.McpServerConfig{
		"fake": {Command: "/nonexistent/binary/that/does/not/exist"},
	}

	result, err := Bootstrap(ctx, servers)
	if err != nil {
		t.Fatalf("Bootstrap should not return error for failing servers: %v", err)
	}
	if result.Manager == nil {
		t.Fatal("Manager should not be nil even when servers fail")
	}
}

// ---------------------------------------------------------------------------
// Manager — unit tests
// ---------------------------------------------------------------------------

func TestManager_AddAndRemoveServer(t *testing.T) {
	mgr := NewManager()
	cfg := types.McpServerConfig{Command: "echo"}

	if err := mgr.AddServer("test", cfg); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	conns := mgr.Connections()
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Name != "test" {
		t.Errorf("Name = %q, want %q", conns[0].Name, "test")
	}

	if err := mgr.RemoveServer("test"); err != nil {
		t.Fatalf("RemoveServer: %v", err)
	}

	conns = mgr.Connections()
	if len(conns) != 0 {
		t.Errorf("expected 0 connections after remove, got %d", len(conns))
	}
}

func TestManager_RemoveNonexistent(t *testing.T) {
	mgr := NewManager()
	err := mgr.RemoveServer("nope")
	if err == nil {
		t.Error("RemoveServer for nonexistent should return error")
	}
}

func TestManager_AddServerOverwrite(t *testing.T) {
	mgr := NewManager()
	cfg1 := types.McpServerConfig{Command: "echo1"}
	cfg2 := types.McpServerConfig{Command: "echo2"}

	mgr.AddServer("srv", cfg1)
	mgr.AddServer("srv", cfg2)

	conns := mgr.Connections()
	if len(conns) != 1 {
		t.Errorf("expected 1 connection after overwrite, got %d", len(conns))
	}
}

// ---------------------------------------------------------------------------
// Manager — Instructions
// ---------------------------------------------------------------------------

func TestManager_Instructions(t *testing.T) {
	mgr := NewManager()

	if len(mgr.AllInstructions()) != 0 {
		t.Error("initial instructions should be empty")
	}

	mgr.SetServerInstructions("server1", "Use this server for file ops.")
	mgr.SetServerInstructions("server2", "Use this for search.")

	instructions := mgr.AllInstructions()
	if len(instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instructions))
	}

	got := mgr.GetServerInstructions("server1")
	if got != "Use this server for file ops." {
		t.Errorf("GetServerInstructions = %q", got)
	}

	got = mgr.GetServerInstructions("nonexistent")
	if got != "" {
		t.Errorf("nonexistent server instructions = %q, want empty", got)
	}
}

func TestManager_EmptyInstructionsSkipped(t *testing.T) {
	mgr := NewManager()
	mgr.SetServerInstructions("empty", "")
	mgr.SetServerInstructions("notempty", "hello")

	instructions := mgr.AllInstructions()
	if len(instructions) != 1 {
		t.Errorf("expected 1 instruction (empty skipped), got %d", len(instructions))
	}
}

func TestManager_FormatInstructionsText(t *testing.T) {
	mgr := NewManager()

	if mgr.FormatInstructionsText() != "" {
		t.Error("empty instructions should return empty text")
	}

	mgr.SetServerInstructions("myserver", "Do things.")
	text := mgr.FormatInstructionsText()

	if text == "" {
		t.Fatal("FormatInstructionsText should not be empty")
	}
	if !containsAll(text, "MCP Server Instructions", "myserver", "Do things.") {
		t.Errorf("FormatInstructionsText = %q, missing expected content", text)
	}
}

// ---------------------------------------------------------------------------
// AllTools on empty Manager
// ---------------------------------------------------------------------------

func TestManager_AllTools_Empty(t *testing.T) {
	mgr := NewManager()
	tools := mgr.AllTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools from empty manager, got %d", len(tools))
	}
}

// ---------------------------------------------------------------------------
// DisconnectAll on empty
// ---------------------------------------------------------------------------

func TestManager_DisconnectAll_Empty(t *testing.T) {
	mgr := NewManager()
	if err := mgr.DisconnectAll(); err != nil {
		t.Errorf("DisconnectAll on empty should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// joinErrors
// ---------------------------------------------------------------------------

func TestJoinErrors(t *testing.T) {
	if joinErrors(nil) != nil {
		t.Error("nil errors should return nil")
	}
	if joinErrors([]error{}) != nil {
		t.Error("empty errors should return nil")
	}
}

// ---------------------------------------------------------------------------
// Bootstrap — multiple servers (one succeeds, one fails)
// ---------------------------------------------------------------------------

func TestBootstrap_MultipleServersMixedResults(t *testing.T) {
	ctx := context.Background()
	servers := map[string]types.McpServerConfig{
		"echo_server": {Command: "echo"},
		"bad_server":  {Command: "/nonexistent/binary"},
	}

	result, err := Bootstrap(ctx, servers)
	if err != nil {
		t.Fatalf("Bootstrap should not fail: %v", err)
	}
	if result.Manager == nil {
		t.Fatal("Manager should not be nil")
	}

	conns := result.Manager.Connections()
	if len(conns) != 2 {
		t.Errorf("expected 2 connections registered, got %d", len(conns))
	}
}

// ---------------------------------------------------------------------------
// Manager — ConnectAll + DisconnectAll round-trip
// ---------------------------------------------------------------------------

func TestManager_ConnectDisconnectRoundTrip(t *testing.T) {
	mgr := NewManager()
	cfg := types.McpServerConfig{Command: "echo"}
	mgr.AddServer("s1", cfg)

	ctx := context.Background()
	_ = mgr.ConnectAll(ctx)
	_ = mgr.DisconnectAll()

	conns := mgr.Connections()
	if len(conns) != 1 {
		t.Errorf("after disconnect all, server still registered, expected 1 connection, got %d", len(conns))
	}
	if len(conns) == 1 && conns[0].Status == "connected" {
		t.Error("server should not be 'connected' after DisconnectAll")
	}
}

// ---------------------------------------------------------------------------
// Manager — AllResources on empty
// ---------------------------------------------------------------------------

func TestManager_AllResources_Empty(t *testing.T) {
	mgr := NewManager()
	resources := mgr.AllResources()
	if len(resources) != 0 {
		t.Errorf("expected 0 resources from empty manager, got %d", len(resources))
	}
}

// ---------------------------------------------------------------------------
// Manager — RefreshTools on empty
// ---------------------------------------------------------------------------

func TestManager_RefreshTools_Empty(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()
	if err := mgr.RefreshTools(ctx); err != nil {
		t.Errorf("RefreshTools on empty should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Manager — multiple instructions
// ---------------------------------------------------------------------------

func TestManager_MultipleInstructions(t *testing.T) {
	mgr := NewManager()
	mgr.SetServerInstructions("s1", "Instruction 1")
	mgr.SetServerInstructions("s2", "Instruction 2")
	mgr.SetServerInstructions("s3", "")

	instructions := mgr.AllInstructions()
	if len(instructions) != 2 {
		t.Errorf("expected 2 non-empty instructions, got %d", len(instructions))
	}

	text := mgr.FormatInstructionsText()
	if !containsAll(text, "s1", "Instruction 1", "s2", "Instruction 2") {
		t.Errorf("FormatInstructionsText missing content: %q", text)
	}
	if contains(text, "s3") {
		t.Error("empty instructions should be excluded from formatted text")
	}
}

// ---------------------------------------------------------------------------
// Manager — overwrite instructions
// ---------------------------------------------------------------------------

func TestManager_OverwriteInstructions(t *testing.T) {
	mgr := NewManager()
	mgr.SetServerInstructions("s1", "old")
	mgr.SetServerInstructions("s1", "new")

	got := mgr.GetServerInstructions("s1")
	if got != "new" {
		t.Errorf("expected overwritten instruction, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
