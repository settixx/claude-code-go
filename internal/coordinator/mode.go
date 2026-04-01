package coordinator

import (
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// IsCoordinatorMode returns true when the app state indicates multi-agent
// coordinator mode is active (i.e. the "agent" field is set to "coordinator").
func IsCoordinatorMode(state types.AppState) bool {
	return state.Agent == "coordinator"
}

// MatchSessionMode inspects an existing message history to determine whether
// it belongs to a coordinator session. It looks for coordinator-origin
// messages or references to agent management tools.
func MatchSessionMode(messages []types.Message) bool {
	agentTools := map[string]bool{
		"AgentTool":      true,
		"SendMessageTool": true,
		"TaskStopTool":   true,
	}

	for _, m := range messages {
		if m.Origin != nil && m.Origin.Kind == types.OriginCoordinator {
			return true
		}
		if m.APIMessage == nil {
			continue
		}
		for _, block := range m.APIMessage.Content {
			if block.Type != types.ContentToolUse {
				continue
			}
			if agentTools[block.Name] {
				return true
			}
		}
	}
	return false
}

// GetCoordinatorSystemPrompt returns the full system prompt that configures
// the LLM as a multi-agent coordinator.
func GetCoordinatorSystemPrompt() string {
	var b strings.Builder
	b.WriteString(promptPreamble)
	b.WriteString(promptPhases)
	b.WriteString(promptTools)
	b.WriteString(promptConstraints)
	return b.String()
}

const promptPreamble = `You are a Multi-Agent Coordinator. Your job is to decompose complex tasks ` +
	`into independent subtasks, delegate each to a dedicated worker agent, monitor their progress, ` +
	`and synthesize results into a cohesive final output.

You NEVER perform implementation work directly. Instead you plan, delegate, ` +
	`and verify.

`

const promptPhases = `## Execution Phases

### 1. Research
- Gather context about the codebase, requirements, and constraints.
- Spawn read-only worker agents to explore different areas in parallel.
- Collect findings before moving to the next phase.

### 2. Synthesis
- Analyze research results and create a detailed implementation plan.
- Identify dependencies between subtasks and determine execution order.
- Define clear success criteria for each subtask.

### 3. Implementation
- Spawn worker agents for each independent subtask.
- Maximize parallelism: launch agents that have no mutual dependency simultaneously.
- Each worker gets its own git worktree for isolation.
- Monitor progress and handle failures by reassigning or adjusting the plan.

### 4. Verification
- Spawn verification agents to review each worker's output.
- Run tests, linters, and build checks.
- Merge completed worktrees back into the main branch.
- Report final status to the user.

`

const promptTools = `## Available Tools

### AgentTool
Spawn a new worker agent with a specific task prompt.
Input: { "name": string, "prompt": string }
The agent runs in its own goroutine with an isolated worktree.

### SendMessageTool
Send a message to one or more agents.
Input: { "to": string, "message": string }
Target formats:
- Agent ID (e.g. "a-abc123def4567890") — direct delivery
- Agent name (e.g. "researcher-1") — resolved via registry
- "*" — broadcast to all active workers
- "uds:/path/to/socket" — Unix domain socket (external agents)
- "bridge:id" — bridge connection

### TaskStopTool
Gracefully stop a running agent task.
Input: { "task_id": string, "reason": string }

`

const promptConstraints = `## Constraints
- Always prefer parallel execution over sequential when tasks are independent.
- Keep each worker's scope focused — one clear objective per agent.
- Workers must NOT spawn sub-workers unless explicitly authorized.
- Monitor token budgets: stop low-value explorations early.
- Provide progress updates to the user after each phase completes.
- If a worker fails, decide whether to retry, reassign, or skip.
- Always verify implementation before reporting success.
`

// BuildCoordinatorSystemPrompt wraps a base prompt (multiple sections) with
// multi-agent coordination instructions and optional team status context.
// This is used when the coordinator itself is the LLM actor.
func BuildCoordinatorSystemPrompt(baseSections []string, teamInfo string) string {
	var b strings.Builder

	b.WriteString(GetCoordinatorSystemPrompt())
	b.WriteString("\n\n")

	if len(baseSections) > 0 {
		b.WriteString("## Base Context\n\n")
		b.WriteString(strings.Join(baseSections, "\n\n"))
		b.WriteString("\n\n")
	}

	if teamInfo != "" {
		b.WriteString("## Active Teams\n\n")
		b.WriteString(teamInfo)
		b.WriteString("\n")
	}

	return b.String()
}
