package query

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// SystemPromptConfig supplies all dynamic values that are interpolated into
// the system prompt.
type SystemPromptConfig struct {
	CWD          string
	OS           string
	Date         string
	Tools        []types.Tool
	CustomPrompt string
	AppendPrompt string
}

// BuildSystemPrompt assembles the full system prompt from identity, environment,
// tool listing, custom additions, and appended sections.
func BuildSystemPrompt(cfg SystemPromptConfig) string {
	var b strings.Builder

	writeIdentity(&b)
	b.WriteByte('\n')
	writeEnvironment(&b, cfg)
	b.WriteByte('\n')
	writeToolCapabilities(&b, cfg.Tools)
	b.WriteByte('\n')
	writeGuidelines(&b)

	if cfg.CustomPrompt != "" {
		b.WriteByte('\n')
		writeSection(&b, "CUSTOM INSTRUCTIONS", cfg.CustomPrompt)
	}
	if cfg.AppendPrompt != "" {
		b.WriteByte('\n')
		writeSection(&b, "ADDITIONAL INSTRUCTIONS", cfg.AppendPrompt)
	}

	return b.String()
}

// DefaultSystemPromptConfig returns a SystemPromptConfig populated with
// sensible runtime defaults.
func DefaultSystemPromptConfig(tools []types.Tool, cwd string) SystemPromptConfig {
	return SystemPromptConfig{
		CWD:   cwd,
		OS:    runtime.GOOS,
		Date:  time.Now().Format("2006-01-02"),
		Tools: tools,
	}
}

func writeIdentity(b *strings.Builder) {
	b.WriteString(`<identity>
You are Ti Code, an interactive CLI-based AI coding assistant developed by Ti Labs.
You are a world-class software engineer with deep expertise across programming
languages, frameworks, design patterns, and best practices.
</identity>`)
}

func writeEnvironment(b *strings.Builder, cfg SystemPromptConfig) {
	b.WriteString("\n<environment>\n")
	fmt.Fprintf(b, "Operating System: %s\n", resolveOS(cfg.OS))
	fmt.Fprintf(b, "Working Directory: %s\n", cfg.CWD)
	fmt.Fprintf(b, "Today's Date: %s\n", cfg.Date)
	b.WriteString("</environment>")
}

func writeToolCapabilities(b *strings.Builder, tools []types.Tool) {
	enabled := filterEnabled(tools)
	if len(enabled) == 0 {
		return
	}

	b.WriteString("\n<tools>\n")
	b.WriteString("You have access to the following tools to help you accomplish tasks:\n\n")

	for _, t := range enabled {
		desc, _ := t.Description(nil)
		schema := t.InputSchema()
		writeToolEntry(b, t.Name(), desc, schema)
	}

	b.WriteString("</tools>")
}

func writeToolEntry(b *strings.Builder, name, desc string, schema types.ToolInputSchema) {
	fmt.Fprintf(b, "### %s\n", name)
	if desc != "" {
		fmt.Fprintf(b, "%s\n", desc)
	}
	if len(schema.Required) > 0 {
		b.WriteString("Required parameters: ")
		b.WriteString(strings.Join(schema.Required, ", "))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func writeGuidelines(b *strings.Builder) {
	b.WriteString(`<guidelines>
- ALWAYS read files before editing to understand context.
- Use the most specific tool for each task.
- Follow existing code style and conventions in the project.
- Explain your reasoning when making non-obvious decisions.
- If a task is ambiguous, ask for clarification.
- For large changes, break work into smaller, verifiable steps.
- After making edits, verify correctness when possible.
- Use context.Context for cancellation and timeout propagation.
- Handle errors explicitly; do not discard them.
</guidelines>`)
}

func writeSection(b *strings.Builder, heading, body string) {
	fmt.Fprintf(b, "<%s>\n%s\n</%s>", strings.ToLower(strings.ReplaceAll(heading, " ", "_")), body, strings.ToLower(strings.ReplaceAll(heading, " ", "_")))
}

func resolveOS(os string) string {
	switch os {
	case "darwin":
		return "macOS (Darwin)"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	default:
		return os
	}
}

func filterEnabled(tools []types.Tool) []types.Tool {
	var out []types.Tool
	for _, t := range tools {
		if t.IsEnabled() {
			out = append(out, t)
		}
	}
	return out
}
