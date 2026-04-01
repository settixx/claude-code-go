package agent

import "strings"

// AgentDefinition describes one registered subagent type and its capabilities.
type AgentDefinition struct {
	AgentType       string
	Description     string
	WhenToUse       string
	Tools           []string // allowlist; empty means all tools
	DisallowedTools []string // denylist applied after allowlist
	Model           string   // default model override
	SystemPrompt    string   // custom system prompt prefix
}

var builtInAgents = []AgentDefinition{
	{
		AgentType:   "explore",
		Description: "Fast codebase exploration agent",
		WhenToUse:   "for exploring codebases, finding files, searching for patterns",
		Tools:       []string{"Glob", "Grep", "FileRead", "Bash"},
		Model:       "fast",
		SystemPrompt: `You are a fast codebase exploration agent. Your job is to quickly find
files, search for patterns, and answer questions about the codebase structure.
Be concise and return only the information requested.`,
	},
	{
		AgentType:   "plan",
		Description: "Planning agent for designing implementation approaches",
		WhenToUse:   "for designing implementation approaches before coding",
		Tools:       []string{"Glob", "Grep", "FileRead", "TodoWrite"},
		SystemPrompt: `You are a planning agent. Analyze requirements, explore the relevant code,
and produce a clear implementation plan. Do not make code changes — only plan.`,
	},
	{
		AgentType:   "code-reviewer",
		Description: "Code review agent for analyzing code quality",
		WhenToUse:   "for reviewing code changes and identifying issues",
		Tools:       []string{"Glob", "Grep", "FileRead"},
		SystemPrompt: `You are a code review agent. Examine the code carefully for bugs,
style issues, security concerns, and improvement opportunities.
Provide specific, actionable feedback.`,
	},
	{
		AgentType:   "verification",
		Description: "Verification agent for testing implementations",
		WhenToUse:   "for verifying implementations work correctly",
		Tools:       []string{"Glob", "Grep", "FileRead", "Bash"},
		SystemPrompt: `You are a verification agent. Run tests, check build outputs, and
confirm that the implementation is correct. Report pass/fail clearly.`,
	},
	{
		AgentType:   "generalPurpose",
		Description: "General purpose agent for complex multi-step tasks",
		WhenToUse:   "for complex multi-step tasks requiring full tool access",
		Tools:       nil, // all tools
		SystemPrompt: `You are a general-purpose coding agent with full tool access.
Break complex tasks into steps and execute them methodically.`,
	},
}

// GetBuiltInAgents returns a copy of all built-in agent definitions.
func GetBuiltInAgents() []AgentDefinition {
	out := make([]AgentDefinition, len(builtInAgents))
	copy(out, builtInAgents)
	return out
}

// FindAgent looks up a built-in agent definition by type name (case-insensitive).
func FindAgent(agentType string) *AgentDefinition {
	lower := strings.ToLower(agentType)
	for i := range builtInAgents {
		if strings.ToLower(builtInAgents[i].AgentType) == lower {
			def := builtInAgents[i]
			return &def
		}
	}
	return nil
}

// IsToolAllowed reports whether the given tool name is permitted by the agent definition.
// An empty Tools list means all tools are allowed (minus DisallowedTools).
func (d *AgentDefinition) IsToolAllowed(toolName string) bool {
	for _, denied := range d.DisallowedTools {
		if strings.EqualFold(denied, toolName) {
			return false
		}
	}
	if len(d.Tools) == 0 {
		return true
	}
	for _, allowed := range d.Tools {
		if strings.EqualFold(allowed, toolName) {
			return true
		}
	}
	return false
}
