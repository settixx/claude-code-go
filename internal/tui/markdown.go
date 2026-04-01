package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func init() {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return
	}
	mdRenderer = r
}

// RenderMarkdown renders markdown text using glamour for rich terminal output.
// Falls back to the basic FormatMarkdown if the glamour renderer is unavailable.
func RenderMarkdown(text string) string {
	if mdRenderer == nil {
		return FormatMarkdown(text)
	}
	out, err := mdRenderer.Render(text)
	if err != nil {
		return FormatMarkdown(text)
	}
	return strings.TrimRight(out, "\n")
}
