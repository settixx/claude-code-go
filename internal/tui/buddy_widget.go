package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BuddyWidget renders a small ASCII companion at the bottom of the TUI.
// Hidden by default; toggle via /buddy command.
type BuddyWidget struct {
	visible  bool
	species  string
	frame    string
	text     string
	maxWidth int
}

// NewBuddyWidget returns a hidden buddy widget defaulting to "duck".
func NewBuddyWidget() BuddyWidget {
	return BuddyWidget{species: "duck"}
}

// SetVisible shows or hides the widget.
func (bw *BuddyWidget) SetVisible(v bool) { bw.visible = v }

// Toggle flips visibility.
func (bw *BuddyWidget) Toggle() { bw.visible = !bw.visible }

// IsVisible reports whether the widget is shown.
func (bw BuddyWidget) IsVisible() bool { return bw.visible }

// SetFrame updates the ASCII art frame to render.
func (bw *BuddyWidget) SetFrame(frame string) { bw.frame = frame }

// SetText updates the reaction text shown beside the sprite.
func (bw *BuddyWidget) SetText(text string) { bw.text = text }

// SetSpecies changes the displayed species label.
func (bw *BuddyWidget) SetSpecies(species string) { bw.species = species }

// SetWidth constrains the widget's maximum render width.
func (bw *BuddyWidget) SetWidth(w int) { bw.maxWidth = w }

// Height returns the rendered height in lines (0 when hidden).
func (bw BuddyWidget) Height() int {
	if !bw.visible {
		return 0
	}
	return lipgloss.Height(bw.View())
}

// View renders the buddy box. Returns "" when hidden.
func (bw BuddyWidget) View() string {
	if !bw.visible || bw.frame == "" {
		return ""
	}

	width := bw.maxWidth
	if width <= 0 {
		width = 40
	}

	spriteLines := strings.Split(bw.frame, "\n")
	textLines := wrapBuddyText(bw.text, width-buddyBoxPadding)

	boxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Faint(true)

	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	var b strings.Builder
	b.WriteString(boxStyle.Render(fmt.Sprintf("┌─ %s ", bw.species)))
	b.WriteString(boxStyle.Render(strings.Repeat("─", max(0, innerWidth-len(bw.species)-2)) + "┐"))
	b.WriteByte('\n')

	for _, line := range spriteLines {
		padded := padRight(line, innerWidth)
		b.WriteString(boxStyle.Render("│ "))
		b.WriteString(padded)
		b.WriteString(boxStyle.Render(" │"))
		b.WriteByte('\n')
	}

	for _, line := range textLines {
		padded := padRight(line, innerWidth)
		b.WriteString(boxStyle.Render("│ "))
		b.WriteString(padded)
		b.WriteString(boxStyle.Render(" │"))
		b.WriteByte('\n')
	}

	b.WriteString(boxStyle.Render("└" + strings.Repeat("─", innerWidth+2) + "┘"))
	return b.String()
}

const buddyBoxPadding = 6

func padRight(s string, width int) string {
	visible := len([]rune(StripANSI(s)))
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func wrapBuddyText(text string, width int) []string {
	if text == "" {
		return nil
	}
	if width <= 0 {
		width = 30
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, Dim(cur))
			cur = w
			continue
		}
		cur += " " + w
	}
	lines = append(lines, Dim(cur))
	return lines
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
