package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/query"
	"github.com/settixx/claude-code-go/internal/types"
)

// EngineFactory creates a query.Engine configured for a specific agent.
type EngineFactory func(ctx context.Context, cfg AgentConfig) (*query.Engine, error)

// AgentConfig holds everything needed to run a single agent session.
type AgentConfig struct {
	AgentID       types.AgentId
	Name          string
	Prompt        string
	Definition    *AgentDefinition
	ModelOverride string
	WorktreePath  string
	ParentCWD     string
	Background    bool
}

// RunAgent executes an agent's conversation loop and returns the final text response.
// For background agents it writes output to a file and returns the file path instead.
func RunAgent(ctx context.Context, cfg AgentConfig, factory EngineFactory) (string, error) {
	slog.Info("agent: starting",
		"id", cfg.AgentID,
		"name", cfg.Name,
		"type", cfg.Definition.AgentType,
		"background", cfg.Background,
	)

	engine, err := factory(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("create engine: %w", err)
	}

	if err := engine.Run(ctx, cfg.Prompt); err != nil {
		return "", fmt.Errorf("agent run: %w", err)
	}

	response := extractFinalResponse(engine.History())

	if !cfg.Background {
		return response, nil
	}

	outPath, err := writeBackgroundOutput(cfg.AgentID, response)
	if err != nil {
		return response, fmt.Errorf("write background output: %w", err)
	}
	return outPath, nil
}

// BuildAgentSystemPrompt assembles the system prompt for the agent by
// prepending the agent-specific prefix to the standard environment block.
func BuildAgentSystemPrompt(cfg AgentConfig) string {
	var b strings.Builder

	b.WriteString(defaultAgentPromptPrefix)
	b.WriteByte('\n')

	if cfg.Definition != nil && cfg.Definition.SystemPrompt != "" {
		b.WriteString(cfg.Definition.SystemPrompt)
		b.WriteByte('\n')
	}

	cwd := cfg.ParentCWD
	if cfg.WorktreePath != "" {
		cwd = cfg.WorktreePath
	}
	fmt.Fprintf(&b, "\n<environment>\nWorking Directory: %s\n</environment>\n", cwd)

	return b.String()
}

// CleanupWorktreeIfClean removes the worktree if no uncommitted changes exist.
func CleanupWorktreeIfClean(path string) {
	if path == "" {
		return
	}
	if err := coordinator.RemoveWorktree(path); err != nil {
		slog.Warn("agent: worktree cleanup failed (may have changes)", "path", path, "error", err)
	}
}

// extractFinalResponse walks the conversation history backwards to find the
// last assistant text block.
func extractFinalResponse(h *query.History) string {
	msgs := h.All()
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg.Type != types.MsgAssistant {
			continue
		}
		text := collectTextBlocks(msg.Content)
		if text != "" {
			return text
		}
	}
	return "(agent produced no text response)"
}

func collectTextBlocks(blocks []types.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == types.ContentText && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func writeBackgroundOutput(agentID types.AgentId, content string) (string, error) {
	dir := filepath.Join(os.TempDir(), "ticode-agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	outPath := filepath.Join(dir, string(agentID)+".md")
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return "", err
	}
	slog.Info("agent: background output written", "path", outPath)
	return outPath, nil
}

const defaultAgentPromptPrefix = `<identity>
You are a subagent of Ti Code, an interactive CLI-based AI coding assistant.
You have been launched to handle a specific task autonomously.
Focus on the task given to you and return a clear, concise response when done.
</identity>`
