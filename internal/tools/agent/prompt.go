package agent

import (
	"fmt"
	"strings"
)

// BuildToolDescription generates the tool description shown to the LLM, listing
// available agent types and their capabilities.
func BuildToolDescription(agents []AgentDefinition, isCoordinator bool) string {
	var b strings.Builder

	b.WriteString("Launch a new agent to handle complex, multi-step tasks autonomously.\n\n")
	b.WriteString("The Task tool launches specialized subagents that run in their own goroutine ")
	b.WriteString("with an independent conversation loop.\n\n")

	writeAvailableAgents(&b, agents)
	b.WriteByte('\n')
	writeUsageNotes(&b, isCoordinator)
	b.WriteByte('\n')
	writeExamples(&b)
	b.WriteByte('\n')
	writeWhenNotToUse(&b)

	return b.String()
}

func writeAvailableAgents(b *strings.Builder, agents []AgentDefinition) {
	b.WriteString("## Available Agent Types\n\n")
	for _, a := range agents {
		fmt.Fprintf(b, "- **%s**: %s. When to use: %s.\n", a.AgentType, a.Description, a.WhenToUse)
		if len(a.Tools) > 0 {
			fmt.Fprintf(b, "  Tools: %s\n", strings.Join(a.Tools, ", "))
		} else {
			b.WriteString("  Tools: all available tools\n")
		}
	}
}

func writeUsageNotes(b *strings.Builder, isCoordinator bool) {
	b.WriteString("## Usage Notes\n\n")
	b.WriteString("- Always include a short description summarizing what the agent will do.\n")
	b.WriteString("- Launch multiple agents concurrently when tasks are independent.\n")
	b.WriteString("- Each agent gets its own conversation context; provide all necessary details in the prompt.\n")
	b.WriteString("- Use `isolation: \"worktree\"` when the agent needs to make code changes in isolation (e.g. parallel experiments).\n")
	b.WriteString("- Use `run_in_background: true` for long-running tasks you don't need to wait for.\n")
	b.WriteString("- The agent's final text response is returned as the tool result.\n")

	if isCoordinator {
		b.WriteString("- As a coordinator, you can launch multiple agents to divide work and aggregate results.\n")
	}
}

func writeExamples(b *strings.Builder) {
	b.WriteString("## Examples\n\n")

	b.WriteString("### Explore the codebase\n")
	b.WriteString("```json\n")
	b.WriteString(`{"prompt": "Find all API endpoint handlers and list their routes", "subagent_type": "explore"}`)
	b.WriteString("\n```\n\n")

	b.WriteString("### Plan a feature\n")
	b.WriteString("```json\n")
	b.WriteString(`{"prompt": "Design the implementation plan for adding OAuth2 support", "subagent_type": "plan"}`)
	b.WriteString("\n```\n\n")

	b.WriteString("### Isolated experiment in a worktree\n")
	b.WriteString("```json\n")
	b.WriteString(`{"prompt": "Try refactoring the auth module to use middleware pattern", "subagent_type": "generalPurpose", "isolation": "worktree"}`)
	b.WriteString("\n```\n\n")

	b.WriteString("### Background task\n")
	b.WriteString("```json\n")
	b.WriteString(`{"prompt": "Run the full test suite and report failures", "subagent_type": "verification", "run_in_background": true}`)
	b.WriteString("\n```\n")
}

func writeWhenNotToUse(b *strings.Builder) {
	b.WriteString("## When NOT to Use\n\n")
	b.WriteString("- Simple, single-step tasks you can do directly with existing tools.\n")
	b.WriteString("- Tasks that only need one file read or one grep search.\n")
	b.WriteString("- When you already have all the information you need.\n")
	b.WriteString("- Purely conversational responses that don't require tool use.\n")
}
