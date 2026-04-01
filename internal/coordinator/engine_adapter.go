package coordinator

import (
	"context"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/types"
)

// QueryRunner abstracts the query.Engine so the coordinator package never
// imports query directly, preventing circular dependencies.
type QueryRunner interface {
	Run(ctx context.Context, userMessage string) error
	LastAssistantText() string
}

// QueryRunnerFactory builds a QueryRunner from the provided configuration.
// The real implementation lives in the wiring layer and calls query.NewEngine.
type QueryRunnerFactory func(cfg QueryRunnerConfig) QueryRunner

// QueryRunnerConfig carries everything the factory needs to construct
// a single-shot query engine for one worker agent.
type QueryRunnerConfig struct {
	Client       interfaces.LLMClient
	ToolExecutor interfaces.ToolExecutor
	Renderer     interfaces.Renderer
	Storage      interfaces.SessionStorage
	SystemPrompt string
	Model        string
	MaxTokens    int
	CWD          string
}

// EngineAdapter bridges coordinator workers to the LLM query layer.
type EngineAdapter struct {
	RunnerFactory QueryRunnerFactory
	ClientFactory func() interfaces.LLMClient
	ToolRegistry  *types.ToolRegistry
	Renderer      interfaces.Renderer
	Storage       interfaces.SessionStorage
	Model         string
	MaxTokens     int
	CWD           string
}

// CreateWorkerRunFunc returns a WorkerRunFunc that drives a full LLM query
// loop for a single worker agent. The systemPrompt is injected by the
// coordinator; toolSubset filters the global registry down to the tools
// this particular agent should have access to.
func (a *EngineAdapter) CreateWorkerRunFunc(systemPrompt string, toolSubset []string) WorkerRunFunc {
	return func(ctx context.Context, w *Worker) error {
		executor := a.buildFilteredExecutor(toolSubset)

		client := a.ClientFactory()
		if client == nil {
			return fmt.Errorf("engine adapter: ClientFactory returned nil")
		}

		runner := a.RunnerFactory(QueryRunnerConfig{
			Client:       client,
			ToolExecutor: executor,
			Renderer:     a.Renderer,
			Storage:      a.Storage,
			SystemPrompt: systemPrompt,
			Model:        a.Model,
			MaxTokens:    a.MaxTokens,
			CWD:          a.resolveCWD(w),
		})

		if err := runner.Run(ctx, w.Prompt); err != nil {
			return fmt.Errorf("engine adapter: query run: %w", err)
		}

		w.mu.Lock()
		w.Result = runner.LastAssistantText()
		w.mu.Unlock()
		return nil
	}
}

// buildFilteredExecutor creates a ToolExecutor that only exposes the tools
// named in subset. An empty subset means "all tools".
func (a *EngineAdapter) buildFilteredExecutor(subset []string) interfaces.ToolExecutor {
	if len(subset) == 0 {
		return &registryExecutor{registry: a.ToolRegistry}
	}

	allowed := make(map[string]bool, len(subset))
	for _, name := range subset {
		allowed[strings.ToLower(name)] = true
	}

	filtered := types.NewToolRegistry()
	for _, t := range a.ToolRegistry.All() {
		if allowed[strings.ToLower(t.Name())] {
			filtered.Register(t)
		}
	}
	return &registryExecutor{registry: filtered}
}

func (a *EngineAdapter) resolveCWD(w *Worker) string {
	if w.WorktreePath != "" {
		return w.WorktreePath
	}
	return a.CWD
}

// registryExecutor adapts a types.ToolRegistry to the interfaces.ToolExecutor
// contract so the coordinator can build one without importing the query package.
type registryExecutor struct {
	registry *types.ToolRegistry
}

func (e *registryExecutor) Register(tool types.Tool) {
	e.registry.Register(tool)
}

func (e *registryExecutor) Execute(ctx context.Context, name string, input map[string]interface{}) (*types.ToolResult, error) {
	t := e.registry.Find(name)
	if t == nil {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t.Call(ctx, input)
}

func (e *registryExecutor) Find(name string) types.Tool {
	return e.registry.Find(name)
}

func (e *registryExecutor) All() []types.Tool {
	return e.registry.All()
}
