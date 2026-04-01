package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// permOption is one of the three permission dialog choices.
type permOption int

const (
	permAllow permOption = iota
	permDeny
	permAlwaysAllow
)

// PermissionDialog is a modal overlay that asks the user to allow or deny a tool invocation.
type PermissionDialog struct {
	tool       string
	input      string
	selected   permOption
	responseCh chan bool
}

// NewPermissionDialog creates a dialog for the given tool request.
func NewPermissionDialog(tool, input string, ch chan bool) *PermissionDialog {
	return &PermissionDialog{
		tool:       tool,
		input:      input,
		selected:   permAllow,
		responseCh: ch,
	}
}

// Update handles arrow-key navigation and Enter to confirm.
// Returns the dialog (nil when resolved) and an optional command.
func (d *PermissionDialog) Update(msg tea.KeyMsg) (*PermissionDialog, tea.Cmd) {
	switch msg.Type {
	case tea.KeyLeft:
		d.selected = clampOption(d.selected - 1)
		return d, nil
	case tea.KeyRight:
		d.selected = clampOption(d.selected + 1)
		return d, nil
	case tea.KeyEnter:
		return d.resolve()
	case tea.KeyEsc:
		d.responseCh <- false
		return nil, nil
	default:
		return d, nil
	}
}

func (d *PermissionDialog) resolve() (*PermissionDialog, tea.Cmd) {
	switch d.selected {
	case permAllow:
		d.responseCh <- true
	case permAlwaysAllow:
		d.responseCh <- true
	default:
		d.responseCh <- false
	}
	return nil, nil
}

// View renders the permission dialog as a centered overlay box.
func (d *PermissionDialog) View(width, height int) string {
	boxWidth := min(60, width-4)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	title := titleStyle.Render("  Permission Required")

	toolLine := lipgloss.NewStyle().Bold(true).Render("  Tool: ") + d.tool
	inputPreview := Truncate(d.input, boxWidth-6)
	inputLine := lipgloss.NewStyle().Faint(true).Render("  " + inputPreview)

	buttons := d.renderButtons()

	content := strings.Join([]string{
		"",
		title,
		"",
		toolLine,
		inputLine,
		"",
		buttons,
		"",
	}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Width(boxWidth).
		Padding(0, 1)

	rendered := box.Render(content)
	return centerOverlay(rendered, width, height)
}

func (d *PermissionDialog) renderButtons() string {
	allow := buttonStyle(permAllow, d.selected, lipgloss.Color("2")).Render(" Allow ")
	deny := buttonStyle(permDeny, d.selected, lipgloss.Color("1")).Render(" Deny ")
	always := buttonStyle(permAlwaysAllow, d.selected, lipgloss.Color("4")).Render(" Always Allow ")
	return "  " + allow + "  " + deny + "  " + always
}

func buttonStyle(opt, selected permOption, color lipgloss.Color) lipgloss.Style {
	base := lipgloss.NewStyle().Foreground(color)
	if opt == selected {
		return base.Bold(true).Reverse(true)
	}
	return base
}

func centerOverlay(box string, termW, termH int) string {
	lines := strings.Split(box, "\n")
	boxH := len(lines)

	topPad := (termH - boxH) / 2
	if topPad < 0 {
		topPad = 0
	}

	var b strings.Builder
	for i := 0; i < topPad; i++ {
		b.WriteByte('\n')
	}

	for _, line := range lines {
		lineW := lipgloss.Width(line)
		leftPad := (termW - lineW) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		b.WriteString(strings.Repeat(" ", leftPad))
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

func clampOption(o permOption) permOption {
	if o < permAllow {
		return permAllow
	}
	if o > permAlwaysAllow {
		return permAlwaysAllow
	}
	return o
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
