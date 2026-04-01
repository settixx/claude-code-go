package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// viewportStyles caches lipgloss styles derived from the active theme.
type viewportStyles struct {
	userHeader      lipgloss.Style
	assistantHeader lipgloss.Style
	systemHeader    lipgloss.Style
	toolHeader      lipgloss.Style
	body            lipgloss.Style
	dimBody         lipgloss.Style
	streamCursor    lipgloss.Style
}

func newViewportStyles() viewportStyles {
	return viewportStyles{
		userHeader:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		assistantHeader: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")),
		systemHeader:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
		toolHeader:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")),
		body:            lipgloss.NewStyle(),
		dimBody:         lipgloss.NewStyle().Faint(true),
		streamCursor:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Blink(true),
	}
}

// RenderMessages produces the complete viewport string from a slice of display messages.
// If streaming is true, the last assistant message shows a blinking cursor.
func RenderMessages(msgs []DisplayMessage, streaming bool, width int) string {
	if len(msgs) == 0 {
		return ""
	}

	styles := newViewportStyles()
	var b strings.Builder

	for i, msg := range msgs {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(renderSingleMessage(msg, styles, width))
	}

	if streaming {
		b.WriteString(styles.streamCursor.Render("█"))
	}

	return b.String()
}

// renderSingleMessage formats one DisplayMessage according to its role.
func renderSingleMessage(msg DisplayMessage, s viewportStyles, width int) string {
	switch msg.Role {
	case "user":
		return renderUserMsg(msg, s, width)
	case "assistant":
		return renderAssistantMsg(msg, s, width)
	case "tool":
		return renderToolMsg(msg, s, width)
	case "system":
		return renderSystemMsg(msg, s, width)
	default:
		return s.dimBody.Render(fmt.Sprintf("[%s] %s", msg.Role, msg.Content))
	}
}

func renderUserMsg(msg DisplayMessage, s viewportStyles, width int) string {
	header := s.userHeader.Render("You")
	body := wrapContent(msg.Content, width-2)
	return header + "\n" + indentBlock(body, 2)
}

func renderAssistantMsg(msg DisplayMessage, s viewportStyles, width int) string {
	header := s.assistantHeader.Render("Ti Code")
	body := FormatMarkdown(msg.Content)
	body = wrapContent(body, width-2)
	return header + "\n" + indentBlock(body, 2)
}

func renderToolMsg(msg DisplayMessage, s viewportStyles, _ int) string {
	header := s.toolHeader.Render("⚡ " + msg.ToolName)
	if msg.Content == "" {
		return header
	}
	preview := Truncate(msg.Content, 200)
	return header + "\n" + s.dimBody.Render("  "+preview)
}

func renderSystemMsg(msg DisplayMessage, s viewportStyles, _ int) string {
	header := s.systemHeader.Render("System")
	return header + "\n" + s.dimBody.Render("  "+msg.Content)
}

// wrapContent wraps text to the given width, preserving existing newlines.
func wrapContent(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	return Wrap(text, width)
}

// indentBlock prepends each line with `n` spaces.
func indentBlock(text string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
