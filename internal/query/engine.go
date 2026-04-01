package query

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/settixx/claude-code-go/internal/config"
	"github.com/settixx/claude-code-go/internal/errors"
	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/storage"
	"github.com/settixx/claude-code-go/internal/types"
)

// ClaudeMDProvider is the subset of config.Provider that the engine needs
// to fetch CLAUDE.md rules. Satisfied by *config.Provider.
type ClaudeMDProvider interface {
	GetClaudeMDRules() *config.ClaudeMDRules
}

// EngineConfig holds the dependencies and options needed to construct an Engine.
type EngineConfig struct {
	// LLMClient handles communication with the language model.
	LLMClient interfaces.LLMClient

	// ToolExecutor manages and runs available tools.
	ToolExecutor interfaces.ToolExecutor

	// StateStore provides access to application state.
	StateStore interfaces.StateStore

	// SessionStorage persists conversation histories.
	SessionStorage interfaces.SessionStorage

	// Renderer handles terminal output.
	Renderer interfaces.Renderer

	// PermissionChecker evaluates tool invocations against the permission policy.
	PermissionChecker interfaces.PermissionChecker

	// SystemPrompt is the static system prompt override. When empty,
	// BuildSystemPrompt generates one from the remaining fields.
	SystemPrompt string

	// CustomPrompt is appended to the system prompt's custom section.
	CustomPrompt string

	// AppendPrompt is appended after everything else.
	AppendPrompt string

	// Model is the LLM model identifier (e.g. "claude-sonnet-4-20250514").
	Model string

	// MaxTokens caps the model's output per response.
	MaxTokens int

	// MaxTurns limits the number of loop iterations (0 = unlimited).
	MaxTurns int

	// Budget configures spending limits; nil means unlimited.
	Budget *BudgetConfig

	// CWD is the working directory reported to the model.
	CWD string

	// ConfigProvider supplies CLAUDE.md rules for system prompt injection.
	ConfigProvider ClaudeMDProvider

	// MemoryDir is the directory to load .claude/memory.md from. Typically CWD.
	MemoryDir string

	// StopHooks are invoked when the model produces a terminal response.
	StopHooks []StopHook

	// InitialMessages seeds the conversation history.
	InitialMessages []types.Message
}

// Engine is the main query orchestrator. It owns the conversation lifecycle:
// build the system prompt, enter the query loop, manage the budget, persist
// the session, and handle cancellation.
type Engine struct {
	cfg     EngineConfig
	budget  *Budget
	history *History
}

// NewEngine creates an Engine from the given configuration.
func NewEngine(cfg EngineConfig) *Engine {
	var b *Budget
	if cfg.Budget != nil {
		b = NewBudget(*cfg.Budget)
	}

	return &Engine{
		cfg:     cfg,
		budget:  b,
		history: NewHistory(cfg.InitialMessages),
	}
}

// Run is the main entry point. It appends the user's message to the history,
// builds the query config, enters the conversation loop, persists the session,
// and returns when the model's response is terminal or an error occurs.
func (e *Engine) Run(ctx context.Context, userMessage string) error {
	startTime := time.Now()
	slog.Info("engine: starting run", "message_len", len(userMessage))

	userMsg := NewUserMessage(userMessage)
	e.history.Append(userMsg)
	e.cfg.Renderer.RenderMessage(userMsg)

	systemPrompt := e.resolveSystemPrompt()
	queryConfig := e.buildQueryConfig(systemPrompt)

	lCfg := loopConfig{
		client:      e.cfg.LLMClient,
		executor:    e.cfg.ToolExecutor,
		renderer:    e.cfg.Renderer,
		permissions: e.cfg.PermissionChecker,
		budget:      e.budget,
		query:       queryConfig,
		maxTurns:    e.cfg.MaxTurns,
		hooks:       e.cfg.StopHooks,
	}

	response, reason, err := runLoop(ctx, lCfg, e.history)
	elapsed := time.Since(startTime)

	slog.Info("engine: run complete",
		"stop_reason", reason,
		"duration", elapsed,
		"turns", e.history.Len(),
		"error", err,
	)

	e.persistSession(response)

	if err != nil {
		if errors.IsAbortError(err) || ctx.Err() != nil {
			return nil
		}
		e.cfg.Renderer.RenderError(err)
		return err
	}
	return nil
}

// History returns the conversation history for external inspection.
func (e *Engine) History() *History {
	return e.history
}

// Budget returns the budget tracker, or nil when unbounded.
func (e *Engine) BudgetTracker() *Budget {
	return e.budget
}

func (e *Engine) resolveSystemPrompt() string {
	if e.cfg.SystemPrompt != "" {
		return e.cfg.SystemPrompt
	}
	spCfg := DefaultSystemPromptConfig(e.cfg.CWD, e.cfg.Model, "")

	customPrompt := e.cfg.CustomPrompt
	if e.cfg.ConfigProvider != nil {
		if rules := e.cfg.ConfigProvider.GetClaudeMDRules(); rules != nil && rules.Content != "" {
			customPrompt = rules.Content + "\n\n" + customPrompt
		}
	}
	spCfg.CustomPrompt = customPrompt
	spCfg.AppendPrompt = e.cfg.AppendPrompt

	if e.cfg.MemoryDir != "" {
		mem, err := storage.LoadMemory(e.cfg.MemoryDir)
		if err != nil {
			slog.Warn("engine: failed to load memory", "dir", e.cfg.MemoryDir, "error", err)
		} else if mem != "" {
			spCfg.MemoryPrompt = mem
		}
	}

	if e.cfg.ToolExecutor != nil {
		enabled := make(map[string]bool)
		for _, t := range e.cfg.ToolExecutor.All() {
			if t.IsEnabled() {
				enabled[t.Name()] = true
			}
		}
		spCfg.EnabledTools = enabled
	}

	return strings.Join(BuildSystemPrompt(spCfg), "\n\n")
}

func (e *Engine) buildQueryConfig(systemPrompt string) types.QueryConfig {
	var toolDefs []types.ToolDef
	if e.cfg.ToolExecutor != nil {
		for _, t := range e.cfg.ToolExecutor.All() {
			if !t.IsEnabled() {
				continue
			}
			desc, _ := t.Description(nil)
			toolDefs = append(toolDefs, types.ToolDef{
				Name:        t.Name(),
				Description: desc,
				InputSchema: t.InputSchema(),
			})
		}
	}

	maxTokens := e.cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16384
	}

	return types.QueryConfig{
		Model:        e.cfg.Model,
		MaxTokens:    maxTokens,
		SystemPrompt: systemPrompt,
		Tools:        toolDefs,
	}
}

func (e *Engine) persistSession(response *types.APIMessage) {
	if e.cfg.SessionStorage == nil {
		return
	}
	state := e.cfg.StateStore.Get()
	sessionID := types.SessionId(state.Agent)
	if sessionID == "" {
		return
	}
	msgs := e.history.All()
	if err := e.cfg.SessionStorage.Save(sessionID, msgs); err != nil {
		slog.Error("engine: failed to persist session", "error", err)
	}
}
