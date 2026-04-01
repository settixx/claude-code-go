package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxInputLines = 5

// InputModel wraps a bubbles textinput with command history, slash-command
// completion, and multi-line editing (Ctrl+J inserts a newline).
type InputModel struct {
	textInput      textinput.Model
	state          InputState
	showSuggestions bool
	suggestions    []string
	selectedSugg   int

	// Multi-line state: extra lines beyond what the single-line textinput holds.
	// lines[0..n-2] are completed lines; the textinput holds the current line.
	lines []string
	width int
}

// InputState tracks history navigation and the current prompt mode.
type InputState struct {
	history    []string
	historyIdx int
	draft      string // preserved partial input while browsing history
	mode       string // "normal", "plan"
}

// slashCommands is the static list used for tab-completion.
var slashCommands = []string{
	"/help", "/exit", "/quit", "/clear", "/model", "/cost",
	"/compact", "/config", "/resume", "/status",
	"/mode", "/session", "/export", "/memory", "/commit",
	"/diff", "/doctor", "/permissions", "/mcp", "/bug",
	"/buddy",
}

// NewInputModel creates an InputModel with styled prompt.
func NewInputModel(mode string) InputModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 4096
	ti.Width = 80

	m := InputModel{
		textInput: ti,
		state: InputState{
			mode:       mode,
			historyIdx: -1,
		},
	}
	m.applyPromptStyle()
	return m
}

// SetWidth adjusts the input width when the terminal resizes.
func (m *InputModel) SetWidth(w int) {
	m.width = w
	m.textInput.Width = w - promptWidth(m.state.mode) - 2
}

// SetMode switches between "normal" and "plan", updating the prompt style.
func (m *InputModel) SetMode(mode string) {
	m.state.mode = mode
	m.applyPromptStyle()
}

// Value returns the full multi-line text (joined with newlines).
func (m *InputModel) Value() string {
	if len(m.lines) == 0 {
		return m.textInput.Value()
	}
	return strings.Join(m.lines, "\n") + "\n" + m.textInput.Value()
}

// SetValue replaces the input text (splits on newlines if multi-line).
func (m *InputModel) SetValue(s string) {
	parts := strings.Split(s, "\n")
	if len(parts) <= 1 {
		m.lines = nil
		m.textInput.SetValue(s)
		return
	}
	m.lines = parts[:len(parts)-1]
	m.textInput.SetValue(parts[len(parts)-1])
}

// Reset clears the input and resets history navigation.
func (m *InputModel) Reset() {
	m.lines = nil
	m.textInput.SetValue("")
	m.state.historyIdx = -1
	m.state.draft = ""
}

// Submit records the current value into history, then resets.
func (m *InputModel) Submit() string {
	val := strings.TrimSpace(m.Value())
	if val != "" {
		m.state.history = append(m.state.history, val)
	}
	m.Reset()
	return val
}

// LineCount returns the number of visible lines in the input area.
func (m *InputModel) LineCount() int {
	return len(m.lines) + 1
}

// Focus delegates to the underlying textinput.
func (m *InputModel) Focus() tea.Cmd { return m.textInput.Focus() }

// Blur delegates to the underlying textinput.
func (m *InputModel) Blur() { m.textInput.Blur() }

// Update handles key events and delegates the rest to the inner textinput.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if cmd, handled := m.handleSpecialKey(keyMsg); handled {
			m.updateSuggestions()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.updateSuggestions()
	return m, cmd
}

// View renders the prompt + input area, plus the suggestion popup when active.
func (m InputModel) View() string {
	prompt := m.promptString()
	contPrompt := strings.Repeat(" ", promptWidth(m.state.mode)-2) + "· "

	var b strings.Builder

	for _, line := range m.lines {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(contPrompt))
		b.WriteString(line)
		b.WriteByte('\n')
	}

	if len(m.lines) > 0 {
		_ = prompt
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(contPrompt))
		b.WriteString(m.textInput.View()[len(m.textInput.Prompt):])
	} else {
		b.WriteString(m.textInput.View())
	}

	if m.showSuggestions && len(m.suggestions) > 0 {
		b.WriteByte('\n')
		b.WriteString(m.renderSuggestions())
	}

	return b.String()
}

// --- internal helpers ---

func (m *InputModel) promptString() string {
	if m.state.mode == "plan" {
		return "plan> "
	}
	return "> "
}

func (m *InputModel) handleSpecialKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlJ:
		m.insertNewline()
		return nil, true
	case tea.KeyUp:
		if m.showSuggestions && len(m.suggestions) > 0 {
			m.selectedSugg = (m.selectedSugg - 1 + len(m.suggestions)) % len(m.suggestions)
			return nil, true
		}
		m.historyPrev()
		return nil, true
	case tea.KeyDown:
		if m.showSuggestions && len(m.suggestions) > 0 {
			m.selectedSugg = (m.selectedSugg + 1) % len(m.suggestions)
			return nil, true
		}
		m.historyNext()
		return nil, true
	case tea.KeyTab:
		if m.showSuggestions && len(m.suggestions) > 0 {
			m.applySuggestion()
			return nil, true
		}
		m.tabComplete()
		return nil, true
	case tea.KeyEscape:
		if m.showSuggestions {
			m.dismissSuggestions()
			return nil, true
		}
		return nil, false
	default:
		return nil, false
	}
}

func (m *InputModel) insertNewline() {
	if m.LineCount() >= maxInputLines {
		return
	}
	m.lines = append(m.lines, m.textInput.Value())
	m.textInput.SetValue("")
}

func (m *InputModel) historyPrev() {
	if len(m.state.history) == 0 {
		return
	}
	if m.state.historyIdx == -1 {
		m.state.draft = m.Value()
		m.state.historyIdx = len(m.state.history) - 1
	} else if m.state.historyIdx > 0 {
		m.state.historyIdx--
	}
	m.SetValue(m.state.history[m.state.historyIdx])
}

func (m *InputModel) historyNext() {
	if m.state.historyIdx == -1 {
		return
	}
	if m.state.historyIdx < len(m.state.history)-1 {
		m.state.historyIdx++
		m.SetValue(m.state.history[m.state.historyIdx])
		return
	}
	m.state.historyIdx = -1
	m.SetValue(m.state.draft)
}

func (m *InputModel) tabComplete() {
	val := m.textInput.Value()
	if !strings.HasPrefix(val, "/") || len(m.lines) > 0 {
		return
	}
	prefix := strings.ToLower(val)
	var matches []string
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	if len(matches) == 1 {
		m.textInput.SetValue(matches[0] + " ")
	}
}

func (m *InputModel) applyPromptStyle() {
	switch m.state.mode {
	case "plan":
		m.textInput.Prompt = Yellow("plan> ")
	default:
		m.textInput.Prompt = Cyan("> ")
	}
}

func promptWidth(mode string) int {
	if mode == "plan" {
		return 6 // "plan> "
	}
	return 2 // "> "
}

// --- suggestion helpers ---

func (m *InputModel) updateSuggestions() {
	val := strings.ToLower(m.textInput.Value())
	if len(m.lines) > 0 || !strings.HasPrefix(val, "/") || len(val) < 2 {
		m.dismissSuggestions()
		return
	}
	matches := filterCommands(val)
	if len(matches) == 0 {
		m.dismissSuggestions()
		return
	}
	m.suggestions = matches
	m.showSuggestions = true
	if m.selectedSugg >= len(matches) {
		m.selectedSugg = 0
	}
}

func (m *InputModel) applySuggestion() {
	if m.selectedSugg >= len(m.suggestions) {
		return
	}
	m.textInput.SetValue(m.suggestions[m.selectedSugg] + " ")
	m.dismissSuggestions()
}

func (m *InputModel) dismissSuggestions() {
	m.showSuggestions = false
	m.suggestions = nil
	m.selectedSugg = 0
}

func (m InputModel) renderSuggestions() string {
	var b strings.Builder
	maxShow := 8
	if len(m.suggestions) < maxShow {
		maxShow = len(m.suggestions)
	}
	for i := 0; i < maxShow; i++ {
		if i == m.selectedSugg {
			b.WriteString(Cyan(" ▸ " + m.suggestions[i]))
		} else {
			b.WriteString(Dim("   " + m.suggestions[i]))
		}
		if i < maxShow-1 {
			b.WriteByte('\n')
		}
	}
	if len(m.suggestions) > maxShow {
		b.WriteString(Dim(fmt.Sprintf("\n   … and %d more", len(m.suggestions)-maxShow)))
	}
	return b.String()
}

func filterCommands(prefix string) []string {
	var matches []string
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}
