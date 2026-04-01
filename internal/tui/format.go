package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// FormatMarkdown applies basic ANSI formatting to markdown-flavored text.
//
// Supported patterns:
//   - **bold** → ANSI bold
//   - `inline code` → ANSI dim
//   - Fenced code blocks (```) → indented with dim color
//   - # Headings → bold + underline
//   - List items (- / *) → preserved with indent
func FormatMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var out strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				out.WriteString(Dim("  ┌─"))
			} else {
				out.WriteString(Dim("  └─"))
			}
			out.WriteByte('\n')
			continue
		}

		if inCodeBlock {
			out.WriteString(Dim("  │ " + line))
			out.WriteByte('\n')
			continue
		}

		formatted := formatMarkdownLine(line)
		out.WriteString(formatted)
		out.WriteByte('\n')
	}

	return strings.TrimRight(out.String(), "\n")
}

func formatMarkdownLine(line string) string {
	trimmed := strings.TrimSpace(line)

	if heading, ok := parseHeading(trimmed); ok {
		return BoldUnderline(heading)
	}

	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return "  " + trimmed[:2] + formatInline(trimmed[2:])
	}

	return formatInline(line)
}

func parseHeading(line string) (string, bool) {
	level := 0
	for _, ch := range line {
		if ch == '#' {
			level++
			continue
		}
		break
	}
	if level == 0 || level > 6 {
		return "", false
	}
	text := strings.TrimSpace(line[level:])
	if text == "" {
		return "", false
	}
	return text, true
}

// formatInline handles **bold** and `code` spans within a single line.
func formatInline(s string) string {
	s = formatBoldSpans(s)
	s = formatCodeSpans(s)
	return s
}

func formatBoldSpans(s string) string {
	var b strings.Builder
	for {
		start := strings.Index(s, "**")
		if start == -1 {
			b.WriteString(s)
			break
		}
		end := strings.Index(s[start+2:], "**")
		if end == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:start])
		b.WriteString(Bold(s[start+2 : start+2+end]))
		s = s[start+2+end+2:]
	}
	return b.String()
}

func formatCodeSpans(s string) string {
	var b strings.Builder
	for {
		start := strings.Index(s, "`")
		if start == -1 {
			b.WriteString(s)
			break
		}
		end := strings.Index(s[start+1:], "`")
		if end == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:start])
		b.WriteString(Dim(s[start+1 : start+1+end]))
		s = s[start+1+end+1:]
	}
	return b.String()
}

// FormatContentBlock renders a single ContentBlock for terminal display.
func FormatContentBlock(block types.ContentBlock) string {
	switch block.Type {
	case types.ContentText:
		return FormatMarkdown(block.Text)
	case types.ContentToolUse:
		input := formatJSONCompact(block.Input)
		return FormatToolUse(block.Name, input)
	case types.ContentToolResult:
		return FormatToolResult(contentBlockText(block))
	case types.ContentThinking:
		return Dim("💭 " + Truncate(block.Thinking, 200))
	default:
		return Dim(fmt.Sprintf("[%s block]", block.Type))
	}
}

func contentBlockText(block types.ContentBlock) string {
	if block.Text != "" {
		return block.Text
	}
	var parts []string
	for _, sub := range block.Content {
		if sub.Text != "" {
			parts = append(parts, sub.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// FormatToolUse renders a tool invocation line.
func FormatToolUse(name, input string) string {
	header := Yellow("⚡ " + Bold(name))
	if input == "" || input == "{}" {
		return header
	}
	truncated := Truncate(input, 200)
	return header + "\n" + Dim("  "+truncated)
}

// FormatToolResult renders tool output, truncating if longer than 500 chars.
func FormatToolResult(result string) string {
	if result == "" {
		return Dim("  (no output)")
	}
	truncated := Truncate(result, 500)
	lines := strings.Split(truncated, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(Dim("  " + line))
	}
	return b.String()
}

func formatJSONCompact(v map[string]interface{}) string {
	if len(v) == 0 {
		return "{}"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
