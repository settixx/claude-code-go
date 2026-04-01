package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel wraps a bubbles textinput with command history and slash-command completion.
type InputModel struct {
	textInput textinput.Model
	state     InputState
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
	m.textInput.Width = w - promptWidth(m.state.mode) - 2
}

// SetMode switches between "normal" and "plan", updating the prompt style.
func (m *InputModel) SetMode(mode string) {
	m.state.mode = mode
	m.applyPromptStyle()
}

// Value returns the current text.
func (m *InputModel) Value() string { return m.textInput.Value() }

// SetValue replaces the input text.
func (m *InputModel) SetValue(s string) { m.textInput.SetValue(s) }

// Reset clears the input and resets history navigation.
func (m *InputModel) Reset() {
	m.textInput.SetValue("")
	m.state.historyIdx = -1
	m.state.draft = ""
}

// Submit records the current value into history, then resets.
func (m *InputModel) Submit() string {
	val := strings.TrimSpace(m.textInput.Value())
	if val != "" {
		m.state.history = append(m.state.history, val)
	}
	m.Reset()
	return val
}

// Focus delegates to the underlying textinput.
func (m *InputModel) Focus() tea.Cmd { return m.textInput.Focus() }

// Blur delegates to the underlying textinput.
func (m *InputModel) Blur() { m.textInput.Blur() }

// Update handles key events and delegates the rest to the inner textinput.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if cmd, handled := m.handleSpecialKey(keyMsg); handled {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the prompt + input.
func (m InputModel) View() string {
	return m.textInput.View()
}

// --- internal helpers ---

func (m *InputModel) handleSpecialKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyUp:
		m.historyPrev()
		return nil, true
	case tea.KeyDown:
		m.historyNext()
		return nil, true
	case tea.KeyTab:
		m.tabComplete()
		return nil, true
	default:
		return nil, false
	}
}

func (m *InputModel) historyPrev() {
	if len(m.state.history) == 0 {
		return
	}
	if m.state.historyIdx == -1 {
		m.state.draft = m.textInput.Value()
		m.state.historyIdx = len(m.state.history) - 1
	} else if m.state.historyIdx > 0 {
		m.state.historyIdx--
	}
	m.textInput.SetValue(m.state.history[m.state.historyIdx])
}

func (m *InputModel) historyNext() {
	if m.state.historyIdx == -1 {
		return
	}
	if m.state.historyIdx < len(m.state.history)-1 {
		m.state.historyIdx++
		m.textInput.SetValue(m.state.history[m.state.historyIdx])
		return
	}
	m.state.historyIdx = -1
	m.textInput.SetValue(m.state.draft)
}

func (m *InputModel) tabComplete() {
	val := m.textInput.Value()
	if !strings.HasPrefix(val, "/") {
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
