package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ANSI escape code constants.
const (
	AnsiReset     = "\033[0m"
	AnsiBold      = "\033[1m"
	AnsiDim       = "\033[2m"
	AnsiItalic    = "\033[3m"
	AnsiUnderline = "\033[4m"

	ansiFgBlack   = "\033[30m"
	ansiFgRed     = "\033[31m"
	ansiFgGreen   = "\033[32m"
	ansiFgYellow  = "\033[33m"
	ansiFgBlue    = "\033[34m"
	ansiFgMagenta = "\033[35m"
	ansiFgCyan    = "\033[36m"
	ansiFgWhite   = "\033[37m"

	AnsiClearLine = "\033[2K"
	AnsiCursorUp  = "\033[1A"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Red wraps s with red foreground ANSI codes.
func Red(s string) string { return ansiFgRed + s + AnsiReset }

// Green wraps s with green foreground ANSI codes.
func Green(s string) string { return ansiFgGreen + s + AnsiReset }

// Yellow wraps s with yellow foreground ANSI codes.
func Yellow(s string) string { return ansiFgYellow + s + AnsiReset }

// Blue wraps s with blue foreground ANSI codes.
func Blue(s string) string { return ansiFgBlue + s + AnsiReset }

// Cyan wraps s with cyan foreground ANSI codes.
func Cyan(s string) string { return ansiFgCyan + s + AnsiReset }

// Magenta wraps s with magenta foreground ANSI codes.
func Magenta(s string) string { return ansiFgMagenta + s + AnsiReset }

// Dim wraps s with dim ANSI codes.
func Dim(s string) string { return AnsiDim + s + AnsiReset }

// Bold wraps s with bold ANSI codes.
func Bold(s string) string { return AnsiBold + s + AnsiReset }

// BoldUnderline wraps s with bold + underline ANSI codes.
func BoldUnderline(s string) string { return AnsiBold + AnsiUnderline + s + AnsiReset }

// StripANSI removes all ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// TermWidth returns the terminal width in columns, defaulting to 80.
func TermWidth() int {
	w, _ := termSize()
	if w <= 0 {
		return 80
	}
	return w
}

// termSize is implemented in platform-specific files:
// colors_unix.go (linux, darwin) and colors_windows.go.

// Truncate shortens s to maxLen runes, appending "…" if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// HorizontalRule returns a dim line of dashes matching terminal width.
func HorizontalRule() string {
	return Dim(strings.Repeat("─", TermWidth()))
}

// Wrap performs basic word-wrap at the given width.
func Wrap(s string, width int) string {
	if width <= 0 {
		width = 80
	}
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if len(line) <= width {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		wrapLine(&b, line, width)
	}
	return strings.TrimRight(b.String(), "\n")
}

func wrapLine(b *strings.Builder, line string, width int) {
	words := strings.Fields(line)
	col := 0
	for i, w := range words {
		wLen := len(StripANSI(w))
		if i > 0 && col+1+wLen > width {
			b.WriteByte('\n')
			col = 0
		} else if i > 0 {
			b.WriteByte(' ')
			col++
		}
		b.WriteString(w)
		col += wLen
	}
	b.WriteByte('\n')
}

// ColorIndex returns a deterministic color for a given index, cycling through
// the palette. Useful for coloring tool names or agent IDs.
func ColorIndex(idx int) func(string) string {
	palette := []func(string) string{Cyan, Green, Yellow, Blue, Magenta, Red}
	return palette[idx%len(palette)]
}

// FormatCost formats a USD amount for display.
func FormatCost(usd float64) string {
	if usd < 0.01 {
		return fmt.Sprintf("$%s", strconv.FormatFloat(usd, 'f', 4, 64))
	}
	return fmt.Sprintf("$%.2f", usd)
}
