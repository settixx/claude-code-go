package coordinator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// AgentSpawnConfig describes how to create a new LLM-backed worker agent.
type AgentSpawnConfig struct {
	Name         string
	Prompt       string
	SystemPrompt string
	ToolSubset   []string
	WorktreePath string
	Background   bool
}

// TeamTask is one subtask inside a team spawn request.
type TeamTask struct {
	Name   string
	Prompt string
	Tools  []string
}

// Orchestrator is the top-level coordinator that manages multi-agent workflows.
// It owns the worker pool, router, team manager, and the engine adapter that
// wires workers to real LLM calls.
type Orchestrator struct {
	Pool          *WorkerPool
	Router        *Router
	TeamManager   *TeamManager
	EngineAdapter *EngineAdapter
}

// NewOrchestrator creates an Orchestrator with all sub-components wired up.
// Pass nil for mailbox if filesystem-based routing is not needed.
func NewOrchestrator(adapter *EngineAdapter, mailbox *Mailbox) *Orchestrator {
	pool := NewWorkerPool()
	return &Orchestrator{
		Pool:          pool,
		Router:        NewRouter(pool, mailbox),
		TeamManager:   NewTeamManager(pool),
		EngineAdapter: adapter,
	}
}

// SpawnAgent creates a new worker backed by a real LLM query loop.
// The worker starts immediately in its own goroutine.
func (o *Orchestrator) SpawnAgent(ctx context.Context, cfg AgentSpawnConfig) (*Worker, error) {
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = GetCoordinatorSystemPrompt()
	}

	runFn := o.EngineAdapter.CreateWorkerRunFunc(systemPrompt, cfg.ToolSubset)

	w, err := o.Pool.SpawnWorker(ctx, cfg.Name, cfg.Prompt, runFn)
	if err != nil {
		return nil, fmt.Errorf("spawn agent %q: %w", cfg.Name, err)
	}

	if cfg.WorktreePath != "" {
		w.mu.Lock()
		w.WorktreePath = cfg.WorktreePath
		w.mu.Unlock()
	}

	slog.Info("orchestrator: agent spawned",
		"name", cfg.Name,
		"id", w.ID,
		"background", cfg.Background,
		"tools", len(cfg.ToolSubset),
	)
	return w, nil
}

// SpawnTeam creates a named team of agents working on related subtasks.
// All agents start in parallel; use the returned Team to wait and collect.
func (o *Orchestrator) SpawnTeam(ctx context.Context, name string, tasks []TeamTask) (*Team, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("spawn team %q: no tasks provided", name)
	}

	ids := make([]types.AgentId, 0, len(tasks))
	for _, task := range tasks {
		w, err := o.SpawnAgent(ctx, AgentSpawnConfig{
			Name:       fmt.Sprintf("%s/%s", name, task.Name),
			Prompt:     task.Prompt,
			ToolSubset: task.Tools,
		})
		if err != nil {
			return nil, fmt.Errorf("spawn team %q task %q: %w", name, task.Name, err)
		}
		ids = append(ids, w.ID)
	}

	team, err := o.TeamManager.Create(name, ids)
	if err != nil {
		return nil, fmt.Errorf("register team %q: %w", name, err)
	}

	slog.Info("orchestrator: team spawned", "team", name, "members", len(ids))
	return team, nil
}

// WaitForAgent blocks until the agent finishes and returns its result text.
func (o *Orchestrator) WaitForAgent(id types.AgentId) (string, error) {
	w, ok := o.Pool.Get(id)
	if !ok {
		return "", fmt.Errorf("agent %s not found", id)
	}

	<-w.Done()

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.Err != nil {
		return "", fmt.Errorf("agent %s failed: %w", id, w.Err)
	}
	return w.Result, nil
}

// SendToAgent routes a text message to a specific agent.
func (o *Orchestrator) SendToAgent(id types.AgentId, msg string) error {
	m := types.Message{
		Type: types.MsgUser,
		Role: "user",
		Text: msg,
		Origin: &types.MessageOrigin{
			Kind: types.OriginCoordinator,
		},
	}
	return o.Router.RouteMessage("coordinator", string(id), m)
}

// WaitForTeam blocks until every member of the named team completes.
// Returns a map of agent-name → result text.
func (o *Orchestrator) WaitForTeam(name string) (map[string]string, error) {
	team, ok := o.TeamManager.Get(name)
	if !ok {
		return nil, fmt.Errorf("team %q not found", name)
	}

	results := make(map[string]string, len(team.Members))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	for _, id := range team.Members {
		wg.Add(1)
		go func(agentID types.AgentId) {
			defer wg.Done()
			result, err := o.WaitForAgent(agentID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			w, _ := o.Pool.Get(agentID)
			key := string(agentID)
			if w != nil {
				key = w.Name
			}
			results[key] = result
		}(id)
	}

	wg.Wait()
	return results, firstErr
}

// Shutdown gracefully stops all workers and cleans up.
func (o *Orchestrator) Shutdown() {
	slog.Info("orchestrator: shutting down")
	o.Pool.StopAll()
}
