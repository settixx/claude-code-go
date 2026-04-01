package agent

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// Tool implements types.Tool for spawning subagents.
type Tool struct {
	toolutil.BaseTool
	pool          *coordinator.WorkerPool
	engineFactory EngineFactory
	parentCWD     string
}

// NewTool creates an AgentTool wired to the given pool and engine factory.
func NewTool(pool *coordinator.WorkerPool, factory EngineFactory, parentCWD string) *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:       "Task",
			ToolAliases:    []string{"Agent", "Subagent"},
			ToolSearchHint: "agent subagent task spawn worker",
		},
		pool:          pool,
		engineFactory: factory,
		parentCWD:     parentCWD,
	}
}

// Description returns the dynamic tool description including available agents.
func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	agents := GetBuiltInAgents()
	return BuildToolDescription(agents, false), nil
}

// InputSchema returns the JSON Schema for the Task tool's input parameters.
func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The task for the agent to perform. Include all necessary context.",
			},
			"subagent_type": map[string]interface{}{
				"type":        "string",
				"description": "Agent type to use: explore, plan, code-reviewer, verification, generalPurpose.",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Optional human-readable name for the agent.",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "A short (3-5 word) description of what the agent will do.",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"description": "Optional model override (e.g. 'fast').",
			},
			"isolation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"default", "worktree"},
				"description": "Isolation mode. 'worktree' creates a git worktree for the agent.",
			},
			"run_in_background": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, run the agent in the background and return immediately.",
			},
		},
		Required: []string{"prompt"},
	}
}

// IsReadOnly returns false — agents can make changes.
func (t *Tool) IsReadOnly(_ map[string]interface{}) bool { return false }

// CheckPermissions requires user approval for agent creation.
func (t *Tool) CheckPermissions(_ map[string]interface{}, mode types.PermissionMode) types.PermissionResult {
	if mode == types.PermBypassPermissions || mode == types.PermAuto {
		return types.PermissionResult{Allowed: true}
	}
	return types.PermissionResult{
		Allowed: false,
		Reason:  "Agent creation requires user approval",
	}
}

// Call executes the agent tool: validates input, spawns a worker, runs the agent loop.
func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	params, err := parseInput(input)
	if err != nil {
		return nil, err
	}

	def := resolveDefinition(params.SubagentType)
	if def == nil {
		return nil, fmt.Errorf("unknown subagent_type %q; valid types: explore, plan, code-reviewer, verification, generalPurpose", params.SubagentType)
	}

	worktreePath, err := maybeCreateWorktree(params.Isolation, t.parentCWD, params.Name)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	agentCfg := AgentConfig{
		Name:          params.Name,
		Prompt:        params.Prompt,
		Definition:    def,
		ModelOverride: resolveModel(params.Model, def.Model),
		WorktreePath:  worktreePath,
		ParentCWD:     t.parentCWD,
		Background:    params.Background,
	}

	factory := t.engineFactory
	runFn := func(ctx context.Context, w *coordinator.Worker) error {
		agentCfg.AgentID = w.ID
		result, runErr := RunAgent(ctx, agentCfg, factory)
		w.Result = result
		return runErr
	}

	worker, err := t.pool.SpawnWorker(ctx, params.Name, params.Prompt, runFn)
	if err != nil {
		cleanupOnFailure(worktreePath)
		return nil, fmt.Errorf("spawn worker: %w", err)
	}
	worker.WorktreePath = worktreePath

	if params.Background {
		return backgroundResult(worker), nil
	}

	<-worker.Done()
	return foregroundResult(worker, worktreePath)
}

// agentInput holds the parsed and validated tool input fields.
type agentInput struct {
	Prompt       string
	SubagentType string
	Name         string
	Description  string
	Model        string
	Isolation    string
	Background   bool
}

func parseInput(input map[string]interface{}) (*agentInput, error) {
	prompt, err := toolutil.RequireString(input, "prompt")
	if err != nil {
		return nil, err
	}

	name := toolutil.OptionalString(input, "name", "")
	subType := toolutil.OptionalString(input, "subagent_type", "generalPurpose")

	if name == "" {
		name = toolutil.OptionalString(input, "description", subType)
	}

	return &agentInput{
		Prompt:       prompt,
		SubagentType: subType,
		Name:         name,
		Description:  toolutil.OptionalString(input, "description", ""),
		Model:        toolutil.OptionalString(input, "model", ""),
		Isolation:    toolutil.OptionalString(input, "isolation", "default"),
		Background:   toolutil.OptionalBool(input, "run_in_background", false),
	}, nil
}

func resolveDefinition(agentType string) *AgentDefinition {
	if agentType == "" {
		agentType = "generalPurpose"
	}
	return FindAgent(agentType)
}

func resolveModel(inputModel, defModel string) string {
	if inputModel != "" {
		return inputModel
	}
	return defModel
}

func maybeCreateWorktree(isolation, parentCWD, name string) (string, error) {
	if isolation != "worktree" {
		return "", nil
	}
	branchName := "agent/" + sanitizeBranchName(name)
	return coordinator.CreateWorktree(parentCWD, branchName)
}

func sanitizeBranchName(s string) string {
	var out []byte
	for i := range len(s) {
		c := s[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+32)
		case c == ' ' || c == '_':
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return "unnamed"
	}
	if len(out) > 30 {
		out = out[:30]
	}
	return string(out)
}

func cleanupOnFailure(worktreePath string) {
	if worktreePath != "" {
		_ = coordinator.RemoveWorktree(worktreePath)
	}
}

func backgroundResult(w *coordinator.Worker) *types.ToolResult {
	return &types.ToolResult{
		Data: fmt.Sprintf(
			"Agent launched in background.\n  ID: %s\n  Name: %s\nUse SendMessage to communicate with this agent.",
			w.ID, w.Name,
		),
	}
}

func foregroundResult(w *coordinator.Worker, worktreePath string) (*types.ToolResult, error) {
	if w.Err != nil {
		CleanupWorktreeIfClean(worktreePath)
		return &types.ToolResult{Data: fmt.Sprintf("Agent failed: %v", w.Err)}, nil
	}

	result := w.Result
	if result == "" {
		result = "(agent produced no output)"
	}

	return &types.ToolResult{Data: result}, nil
}
