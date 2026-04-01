package tui

import (
	"strings"
)

// StatusLine represents a three-section status bar rendered at a fixed terminal width.
// Each section (Left, Center, Right) can contain arbitrary text with ANSI colors.
type StatusLine struct {
	// Left is the left-aligned section content.
	Left string
	// Center is the center-aligned section content.
	Center string
	// Right is the right-aligned section content.
	Right string
}

// Render formats the status line to exactly the given width.
// Content is distributed as: left-aligned | centered | right-aligned,
// with padding inserted to fill the line. If width is too narrow,
// sections are truncated from center first, then right.
func (sl *StatusLine) Render(width int) string {
	if width <= 0 {
		return ""
	}

	leftLen := visibleLen(sl.Left)
	centerLen := visibleLen(sl.Center)
	rightLen := visibleLen(sl.Right)

	totalContent := leftLen + centerLen + rightLen
	if totalContent >= width {
		return sl.renderTruncated(width)
	}

	remaining := width - totalContent
	leftPad, rightPad := distributePadding(remaining, leftLen, centerLen, rightLen, width)

	var b strings.Builder
	b.WriteString(sl.Left)
	b.WriteString(strings.Repeat(" ", leftPad))
	b.WriteString(sl.Center)
	b.WriteString(strings.Repeat(" ", rightPad))
	b.WriteString(sl.Right)
	return b.String()
}

func (sl *StatusLine) renderTruncated(width int) string {
	leftLen := visibleLen(sl.Left)
	rightLen := visibleLen(sl.Right)

	if leftLen >= width {
		return Truncate(StripANSI(sl.Left), width)
	}

	spaceForRight := width - leftLen - 1
	if spaceForRight <= 0 {
		return sl.Left + strings.Repeat(" ", width-leftLen)
	}

	right := sl.Right
	if rightLen > spaceForRight {
		right = Truncate(StripANSI(right), spaceForRight)
		rightLen = visibleLen(right)
	}

	gap := width - leftLen - rightLen
	if gap < 0 {
		gap = 0
	}
	return sl.Left + strings.Repeat(" ", gap) + right
}

func distributePadding(remaining, leftLen, centerLen, rightLen, width int) (int, int) {
	if centerLen == 0 {
		return 0, remaining
	}
	idealCenter := (width - centerLen) / 2
	leftPad := idealCenter - leftLen
	if leftPad < 1 {
		leftPad = 1
	}
	rightPad := width - leftLen - leftPad - centerLen - rightLen
	if rightPad < 1 {
		rightPad = 1
	}
	return leftPad, rightPad
}

func visibleLen(s string) int {
	return len([]rune(StripANSI(s)))
}

// StatusLineFromState builds a StatusLine from common session parameters.
func StatusLineFromState(model string, ct *CostTracker, permMode, sessionID, branch string) *StatusLine {
	left := buildLeftSection(model, permMode, branch)
	center := buildCenterTokens(ct)
	right := buildRightSection(sessionID)
	return &StatusLine{Left: left, Center: center, Right: right}
}

func buildLeftSection(model, permMode, branch string) string {
	parts := make([]string, 0, 3)
	if model != "" {
		parts = append(parts, Bold(model))
	}
	if permMode != "" {
		parts = append(parts, permModeLabel(permMode))
	}
	if branch != "" {
		parts = append(parts, Dim("["+branch+"]"))
	}
	return strings.Join(parts, " ")
}

func buildCenterTokens(ct *CostTracker) string {
	if ct == nil || ct.TotalTokens() == 0 {
		return ""
	}
	return Dim(ct.FormatStatusSegment())
}

func buildRightSection(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	display := sessionID
	if len(display) > 8 {
		display = display[:8]
	}
	return Dim(display)
}

func permModeLabel(mode string) string {
	switch mode {
	case "plan":
		return Blue("[plan]")
	case "auto", "bypassPermissions", "dontAsk":
		return Yellow("[" + mode + "]")
	case "acceptEdits":
		return Green("[accept-edits]")
	default:
		return Dim("[" + mode + "]")
	}
}
