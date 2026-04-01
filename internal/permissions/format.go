package permissions

import (
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/tui"
)

// FormatPermissionRequest renders a boxed permission prompt for terminal display.
func FormatPermissionRequest(req PermissionRequest) string {
	width := tui.TermWidth()
	if width > 60 {
		width = 60
	}
	if width < 30 {
		width = 30
	}
	innerW := width - 4 // account for "│ " and " │"

	var b strings.Builder

	writeTop(&b, width, "Permission Required")
	writeRow(&b, innerW, tui.Bold("Tool:")+"  "+colorToolName(req.ToolName))
	writeRow(&b, innerW, tui.Bold("Risk:")+"  "+colorRisk(req.Risk))

	if req.Description != "" {
		writeEmpty(&b, innerW)
		for _, line := range wrapText(req.Description, innerW) {
			writeRow(&b, innerW, line)
		}
	}

	detail := inputDetail(req.Input)
	if detail != "" {
		writeEmpty(&b, innerW)
		for _, line := range wrapText(detail, innerW) {
			writeRow(&b, innerW, tui.Dim(line))
		}
	}

	writeEmpty(&b, innerW)
	writeRow(&b, innerW, tui.Dim("[a] Allow  [d] Deny  [A] Always Allow  [D] Always Deny"))
	writeBottom(&b, width)

	return b.String()
}

func writeTop(b *strings.Builder, width int, title string) {
	titlePart := "─ " + title + " "
	remaining := width - 2 - len(titlePart)
	if remaining < 1 {
		remaining = 1
	}
	b.WriteString(tui.Cyan("┌" + titlePart + strings.Repeat("─", remaining) + "┐"))
	b.WriteByte('\n')
}

func writeBottom(b *strings.Builder, width int) {
	b.WriteString(tui.Cyan("└" + strings.Repeat("─", width-2) + "┘"))
}

func writeRow(b *strings.Builder, innerW int, text string) {
	visible := len(tui.StripANSI(text))
	pad := innerW - visible
	if pad < 0 {
		pad = 0
	}
	b.WriteString(tui.Cyan("│") + " " + text + strings.Repeat(" ", pad) + " " + tui.Cyan("│"))
	b.WriteByte('\n')
}

func writeEmpty(b *strings.Builder, innerW int) {
	b.WriteString(tui.Cyan("│") + strings.Repeat(" ", innerW+2) + tui.Cyan("│"))
	b.WriteByte('\n')
}

func colorToolName(name string) string {
	return tui.Yellow(tui.Bold(name))
}

func colorRisk(risk string) string {
	switch risk {
	case "high":
		return tui.Red(tui.Bold("HIGH"))
	case "medium":
		return tui.Yellow("medium")
	default:
		return tui.Green("low")
	}
}

func inputDetail(input map[string]interface{}) string {
	keys := []string{"command", "cmd", "file_path", "path", "query", "pattern"}
	var parts []string
	for _, k := range keys {
		v, ok := input[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		display := tui.Truncate(s, 120)
		parts = append(parts, fmt.Sprintf("%s: %s", k, display))
	}
	return strings.Join(parts, "\n")
}

func wrapText(s string, width int) []string {
	var result []string
	for _, raw := range strings.Split(s, "\n") {
		if len(tui.StripANSI(raw)) <= width {
			result = append(result, raw)
			continue
		}
		words := strings.Fields(raw)
		var line strings.Builder
		col := 0
		for _, w := range words {
			wLen := len(tui.StripANSI(w))
			if col > 0 && col+1+wLen > width {
				result = append(result, line.String())
				line.Reset()
				col = 0
			}
			if col > 0 {
				line.WriteByte(' ')
				col++
			}
			line.WriteString(w)
			col += wLen
		}
		if line.Len() > 0 {
			result = append(result, line.String())
		}
	}
	return result
}
