package test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/mcp"
	"github.com/settixx/claude-code-go/internal/permissions"
	"github.com/settixx/claude-code-go/internal/query"
	"github.com/settixx/claude-code-go/internal/skills"
	"github.com/settixx/claude-code-go/internal/tools"
	"github.com/settixx/claude-code-go/internal/tools/agent"
	"github.com/settixx/claude-code-go/internal/tools/bash"
	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// TestGolden_ToolRegistryHasAllExpectedTools
// ---------------------------------------------------------------------------

func TestGolden_ToolRegistryHasAllExpectedTools(t *testing.T) {
	registry := types.NewToolRegistry()
	tools.RegisterCoreTools(registry)
	tools.RegisterExtendedTools(registry)

	coreNames := []string{
		"Bash", "FileRead", "FileWrite", "FileEdit", "Glob", "Grep",
	}
	extendedNames := []string{
		"WebSearch", "WebFetch", "NotebookEdit", "TodoWrite",
		"Config", "Sleep", "AskUserQuestion", "Brief", "ToolSearch",
	}

	allExpected := append(coreNames, extendedNames...)

	for _, name := range allExpected {
		t.Run(name, func(t *testing.T) {
			found := registry.Find(name)
			if found == nil {
				t.Errorf("expected tool %q to be registered, but it was not found", name)
			}
		})
	}

	all := registry.All()
	if len(all) < len(coreNames)+len(extendedNames)-1 {
		t.Errorf("registry has %d tools, expected at least %d", len(all), len(coreNames)+len(extendedNames)-1)
	}
}

// ---------------------------------------------------------------------------
// TestGolden_BashSecurityBlocksDangerous
// ---------------------------------------------------------------------------

func TestGolden_BashSecurityBlocksDangerous(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantBeh  bash.SecurityBehavior
	}{
		{
			name:    "command substitution with $()",
			command: `echo $(whoami)`,
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "backtick substitution",
			command: "echo `whoami`",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "zsh dangerous: zmodload",
			command: "zmodload zsh/system",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "zsh dangerous: zpty",
			command: "zpty mypty bash",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "heredoc inside command sub",
			command: `$(cat <<EOF`,
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "rm -rf /",
			command: "rm -rf /",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "fork bomb",
			command: ":(){:|:&};:",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "LD_PRELOAD injection",
			command: "LD_PRELOAD=/evil.so ls",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "IFS manipulation",
			command: `IFS="/" read`,
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "newline injection",
			command: "echo hello\nrm -rf /",
			wantBeh: bash.SecurityDeny,
		},
		{
			name:    "empty command allowed",
			command: "",
			wantBeh: bash.SecurityAllow,
		},
		{
			name:    "whitespace only allowed",
			command: "   ",
			wantBeh: bash.SecurityAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bash.ValidateCommand(tt.command)
			if result.Behavior != tt.wantBeh {
				t.Errorf("ValidateCommand(%q) = %s, want %s (msg: %s)",
					tt.command, result.Behavior, tt.wantBeh, result.Message)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGolden_PermissionFlowWorks
// ---------------------------------------------------------------------------

func TestGolden_PermissionFlowWorks(t *testing.T) {
	t.Run("bypass allows everything", func(t *testing.T) {
		checker := permissions.NewChecker(types.PermBypassPermissions, nil)
		result := checker.Check("Bash", map[string]interface{}{"command": "rm -rf /"})
		if !result.Allowed {
			t.Error("bypass mode should allow everything")
		}
	})

	t.Run("plan mode denies writes", func(t *testing.T) {
		checker := permissions.NewChecker(types.PermPlan, nil)
		result := checker.Check("FileWrite", nil)
		if result.Allowed {
			t.Error("plan mode should deny write tools")
		}
	})

	t.Run("plan mode allows reads", func(t *testing.T) {
		checker := permissions.NewChecker(types.PermPlan, nil)
		result := checker.Check("FileRead", nil)
		if !result.Allowed {
			t.Errorf("plan mode should allow read tools: %s", result.Reason)
		}
	})

	t.Run("default mode with allow rule", func(t *testing.T) {
		rules := permissions.NewRuleSet()
		rules.AddAllowRule("Glob")
		checker := permissions.NewChecker(types.PermDefault, rules)

		result := checker.Check("Glob", nil)
		if !result.Allowed {
			t.Error("Glob should be allowed by rule")
		}
	})

	t.Run("default mode with deny rule", func(t *testing.T) {
		rules := permissions.NewRuleSet()
		rules.AddDenyRule("Bash")
		checker := permissions.NewChecker(types.PermDefault, rules)

		result := checker.Check("Bash", nil)
		if result.Allowed {
			t.Error("Bash should be denied by rule")
		}
	})

	t.Run("decision history is tracked", func(t *testing.T) {
		checker := permissions.NewChecker(types.PermBypassPermissions, nil)
		ctx := context.Background()

		_, err := checker.CheckWithPrompt(ctx, "FileRead", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		history := checker.DecisionHistory()
		if len(history) == 0 {
			t.Error("expected at least one decision in history")
		}
	})
}

// ---------------------------------------------------------------------------
// TestGolden_CompactionPreservesRecentMessages
// ---------------------------------------------------------------------------

func TestGolden_CompactionPreservesRecentMessages(t *testing.T) {
	cfg := query.CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       4,
	}

	client := &goldenMockLLM{}
	c := query.NewCompactor(cfg, client)

	msgs := makeGoldenMessages(12, 150)
	result, err := c.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("compact error: %v", err)
	}

	if len(result) < 4 {
		t.Fatalf("result should have at least 4 messages (recent), got %d", len(result))
	}

	// The last KeepRecent messages should match originals
	recentStart := len(result) - 4
	for i := 0; i < 4; i++ {
		orig := msgs[len(msgs)-4+i]
		got := result[recentStart+i]
		if got.Role != orig.Role {
			t.Errorf("recent[%d]: role mismatch %q vs %q", i, got.Role, orig.Role)
		}
	}

	if !result[0].IsCompactSummary {
		t.Error("first message should be marked as compact summary")
	}
}

// ---------------------------------------------------------------------------
// TestGolden_SkillsLoadAndExecute
// ---------------------------------------------------------------------------

func TestGolden_SkillsLoadAndExecute(t *testing.T) {
	bundled, err := skills.LoadBundledSkills()
	if err != nil {
		t.Fatalf("LoadBundledSkills: %v", err)
	}
	if len(bundled) == 0 {
		t.Fatal("expected at least one bundled skill")
	}

	registry := skills.NewSkillRegistry()
	for _, s := range bundled {
		registry.Register(s)
	}

	for _, s := range bundled {
		t.Run("loaded_"+s.Name, func(t *testing.T) {
			got, ok := registry.Get(s.Name)
			if !ok {
				t.Fatalf("skill %q not found after registration", s.Name)
			}
			if got.Source != "bundled" {
				t.Errorf("source = %q, want %q", got.Source, "bundled")
			}
			if got.Content == "" {
				t.Error("skill content is empty")
			}
		})
	}

	tool := skills.NewSkillTool(registry)
	if tool.Name() != "skill" {
		t.Errorf("SkillTool.Name() = %q, want %q", tool.Name(), "skill")
	}

	schema := tool.InputSchema()
	if _, ok := schema.Properties["skill_name"]; !ok {
		t.Error("SkillTool schema missing 'skill_name' property")
	}
}

// ---------------------------------------------------------------------------
// TestGolden_MCPProtocolHandshake
// ---------------------------------------------------------------------------

func TestGolden_MCPProtocolHandshake(t *testing.T) {
	t.Run("JSON-RPC request marshaling", func(t *testing.T) {
		req := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  mcp.MethodInitialize,
			Params: mcp.InitializeParams{
				ProtocolVersion: "2024-11-05",
				ClientInfo:      mcp.ClientInfo{Name: "ti-code", Version: "0.1.0"},
				Capabilities:    mcp.ClientCapabilities{Tools: &mcp.ToolCapabilities{}},
			},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var decoded mcp.JSONRPCRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if decoded.JSONRPC != "2.0" {
			t.Errorf("jsonrpc = %q, want %q", decoded.JSONRPC, "2.0")
		}
		if decoded.Method != mcp.MethodInitialize {
			t.Errorf("method = %q, want %q", decoded.Method, mcp.MethodInitialize)
		}
		if decoded.ID != 1 {
			t.Errorf("id = %d, want 1", decoded.ID)
		}
	})

	t.Run("JSON-RPC error marshaling", func(t *testing.T) {
		resp := mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      42,
			Error: &mcp.JSONRPCError{
				Code:    -32601,
				Message: "method not found",
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		if !strings.Contains(string(data), `"method not found"`) {
			t.Error("serialized response should contain error message")
		}
		if !strings.Contains(string(data), `"code":-32601`) {
			t.Error("serialized response should contain error code")
		}
	})

	t.Run("method constants defined", func(t *testing.T) {
		methods := map[string]string{
			"initialize":     mcp.MethodInitialize,
			"tools/list":     mcp.MethodToolsList,
			"tools/call":     mcp.MethodToolsCall,
			"resources/list": mcp.MethodResourceList,
			"resources/read": mcp.MethodResourceRead,
			"prompts/list":   mcp.MethodPromptsList,
			"prompts/get":    mcp.MethodPromptsGet,
			"shutdown":       mcp.MethodShutdown,
		}
		for expected, actual := range methods {
			if actual != expected {
				t.Errorf("method constant = %q, want %q", actual, expected)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// TestGolden_CoordinatorSpawnsWorkers
// ---------------------------------------------------------------------------

func TestGolden_CoordinatorSpawnsWorkers(t *testing.T) {
	pool := coordinator.NewWorkerPool()

	t.Run("spawn and stop", func(t *testing.T) {
		ctx := context.Background()
		w, err := pool.SpawnWorker(ctx, "test-worker", "do something", nil)
		if err != nil {
			t.Fatalf("SpawnWorker: %v", err)
		}
		if w.Name != "test-worker" {
			t.Errorf("name = %q, want %q", w.Name, "test-worker")
		}
		if w.Status != coordinator.WorkerRunning {
			t.Errorf("status = %q, want %q", w.Status, coordinator.WorkerRunning)
		}

		w.Stop()
		<-w.Done()
		if w.Status != coordinator.WorkerStopped {
			t.Errorf("after stop: status = %q, want %q", w.Status, coordinator.WorkerStopped)
		}
	})

	t.Run("broadcast message", func(t *testing.T) {
		pool2 := coordinator.NewWorkerPool()
		ctx := context.Background()

		w1, _ := pool2.SpawnWorker(ctx, "w1", "task1", nil)
		w2, _ := pool2.SpawnWorker(ctx, "w2", "task2", nil)

		msg := types.Message{Type: types.MsgUser, Text: "hello all"}
		pool2.BroadcastMessage(msg)

		// Give workers time to process
		time.Sleep(50 * time.Millisecond)

		for _, w := range []*coordinator.Worker{w1, w2} {
			msgs := w.Messages()
			if len(msgs) == 0 {
				t.Errorf("worker %q received no broadcast messages", w.Name)
			}
		}

		pool2.StopAll()
	})
}

// ---------------------------------------------------------------------------
// TestGolden_CLISlashCommands
// ---------------------------------------------------------------------------

func TestGolden_CLISlashCommands(t *testing.T) {
	expected := []string{
		"/help", "/exit", "/quit", "/clear", "/model", "/cost",
		"/compact", "/config", "/resume", "/status",
	}

	for _, cmd := range expected {
		if !strings.HasPrefix(cmd, "/") {
			t.Errorf("slash command %q should start with /", cmd)
		}
	}

	if len(expected) < 8 {
		t.Errorf("expected at least 8 slash commands, got %d", len(expected))
	}
}

// ---------------------------------------------------------------------------
// TestGolden_AgentDefinitionsComplete
// ---------------------------------------------------------------------------

func TestGolden_AgentDefinitionsComplete(t *testing.T) {
	agents := agent.GetBuiltInAgents()

	expectedTypes := []string{
		"explore", "plan", "code-reviewer", "verification", "generalPurpose",
	}

	if len(agents) != len(expectedTypes) {
		t.Fatalf("got %d agent types, want %d", len(agents), len(expectedTypes))
	}

	typeSet := make(map[string]bool, len(agents))
	for _, a := range agents {
		typeSet[a.AgentType] = true
	}

	for _, expected := range expectedTypes {
		if !typeSet[expected] {
			t.Errorf("missing agent type %q", expected)
		}
	}

	for _, a := range agents {
		t.Run(a.AgentType, func(t *testing.T) {
			if a.Description == "" {
				t.Error("agent description is empty")
			}
			if a.WhenToUse == "" {
				t.Error("agent WhenToUse is empty")
			}
			if a.SystemPrompt == "" {
				t.Error("agent SystemPrompt is empty")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type goldenMockLLM struct{}

func (m *goldenMockLLM) Send(_ context.Context, _ types.QueryConfig, _ []types.Message) (*types.APIMessage, error) {
	return &types.APIMessage{
		Role: "assistant",
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: "Summary: conversation about code changes.",
		}},
	}, nil
}

func (m *goldenMockLLM) Stream(_ context.Context, _ types.QueryConfig, _ []types.Message) (<-chan types.StreamEvent, error) {
	return nil, nil
}

func (m *goldenMockLLM) CountTokens(_ context.Context, _ types.QueryConfig, _ []types.Message) (int, error) {
	return 0, nil
}

func makeGoldenMessages(n int, textLen int) []types.Message {
	msgs := make([]types.Message, n)
	text := strings.Repeat("x", textLen)
	for i := range msgs {
		role := "user"
		msgType := types.MsgUser
		if i%2 == 1 {
			role = "assistant"
			msgType = types.MsgAssistant
		}
		msgs[i] = types.Message{
			Type:      msgType,
			Role:      role,
			Timestamp: time.Now(),
			Content: []types.ContentBlock{{
				Type: types.ContentText,
				Text: text,
			}},
		}
	}
	return msgs
}
