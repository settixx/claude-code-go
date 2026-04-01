package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppModel is the top-level Bubble Tea model for the Ti Code TUI.
type AppModel struct {
	// Input
	textInput InputModel

	// Viewport for scrollable message display
	viewport viewport.Model

	// Loading spinner
	spinner spinner.Model

	// State
	messages  []DisplayMessage
	streaming bool
	streamBuf strings.Builder
	width     int
	height    int
	ready     bool

	// Welcome text shown once at the top
	welcome string

	// Callback for sending user input to the query engine
	onSubmit func(input string)

	// Permission dialog (nil when inactive)
	permDialog *PermissionDialog

	// Status bar state
	modelName  string
	permMode   string
	tokens     int
	cost       float64
	sessionID  string
}

// NewAppModel creates the initial Bubble Tea model.
func NewAppModel(welcome string, onSubmit func(string)) AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	return AppModel{
		textInput: NewInputModel("normal"),
		spinner:   sp,
		welcome:   welcome,
		onSubmit:  onSubmit,
	}
}

// SetStatus updates the status bar fields.
func (m *AppModel) SetStatus(model, permMode, sessionID string, tokens int, cost float64) {
	m.modelName = model
	m.permMode = permMode
	m.sessionID = sessionID
	m.tokens = tokens
	m.cost = cost
}

// Init returns the initial command (text input blink + spinner tick).
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.textInput.Focus(), m.spinner.Tick)
}

// Update is the main event dispatcher.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	case StreamChunkMsg:
		return m.handleStreamChunk(msg), nil
	case StreamDoneMsg:
		return m.handleStreamDone(), nil
	case ToolCallMsg:
		return m.handleToolCall(msg), nil
	case ToolResultMsg:
		return m.handleToolResult(msg), nil
	case PermissionRequestMsg:
		m.permDialog = NewPermissionDialog(msg.Tool, msg.Input, msg.ResponseCh)
		return m, nil
	case ErrorMsg:
		return m.handleError(msg), nil
	}

	return m.updateChildren(msg)
}

// View composes the full screen layout.
func (m AppModel) View() string {
	if !m.ready {
		return "\n  Initializing…"
	}

	if m.permDialog != nil {
		return m.permDialog.View(m.width, m.height)
	}

	header := m.renderHeader()
	status := m.renderStatusLine()
	input := m.textInput.View()
	vpView := m.viewport.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		vpView,
		status,
		input,
	)
}

// --- resize ---

func (m AppModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	headerH := lipgloss.Height(m.renderHeader())
	statusH := 1
	inputH := 1
	vpHeight := m.height - headerH - statusH - inputH
	if vpHeight < 1 {
		vpHeight = 1
	}

	if !m.ready {
		m.viewport = viewport.New(m.width, vpHeight)
		m.viewport.YPosition = headerH
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
	}

	m.textInput.SetWidth(m.width)
	m.refreshViewport()
	return m, nil
}

// --- key handling ---

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.permDialog != nil {
		dialog, cmd := m.permDialog.Update(msg)
		m.permDialog = dialog
		return m, cmd
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEnter:
		return m.submitInput()
	default:
		return m.updateInput(msg)
	}
}

func (m AppModel) submitInput() (tea.Model, tea.Cmd) {
	val := m.textInput.Submit()
	if val == "" {
		return m, nil
	}

	if handled, quit := m.handleSlashCommand(val); handled {
		if quit {
			return m, tea.Quit
		}
		m.refreshViewport()
		return m, nil
	}

	m.messages = append(m.messages, DisplayMessage{Role: "user", Content: val})
	m.streaming = true
	m.streamBuf.Reset()
	m.refreshViewport()

	if m.onSubmit != nil {
		m.onSubmit(val)
	}
	return m, nil
}

func (m *AppModel) handleSlashCommand(line string) (handled bool, quit bool) {
	if !strings.HasPrefix(line, "/") {
		return false, false
	}

	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/exit", "/quit":
		return true, true
	case "/help":
		m.appendSystemMsg(slashHelpText())
		return true, false
	case "/clear":
		m.messages = nil
		return true, false
	default:
		m.appendSystemMsg("Unknown command: " + cmd + ". Type /help for available commands.")
		return true, false
	}
}

func slashHelpText() string {
	cmds := []struct{ cmd, desc string }{
		{"/help", "Show this help message"},
		{"/exit", "Exit the application"},
		{"/quit", "Exit the application"},
		{"/clear", "Clear the screen"},
	}
	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, c := range cmds {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", c.cmd, c.desc))
	}
	return b.String()
}

func (m AppModel) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// --- spinner ---

func (m AppModel) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// --- stream events ---

func (m AppModel) handleStreamChunk(msg StreamChunkMsg) AppModel {
	m.streamBuf.WriteString(msg.Text)
	lastIdx := len(m.messages) - 1
	if lastIdx >= 0 && m.messages[lastIdx].Role == "assistant" {
		m.messages[lastIdx].Content = m.streamBuf.String()
	} else {
		m.messages = append(m.messages, DisplayMessage{
			Role:    "assistant",
			Content: m.streamBuf.String(),
		})
	}
	m.refreshViewport()
	return m
}

func (m AppModel) handleStreamDone() AppModel {
	m.streaming = false
	m.streamBuf.Reset()
	m.refreshViewport()
	return m
}

func (m AppModel) handleToolCall(msg ToolCallMsg) AppModel {
	m.messages = append(m.messages, DisplayMessage{
		Role:     "tool",
		ToolName: msg.Name,
		Content:  msg.Input,
	})
	m.refreshViewport()
	return m
}

func (m AppModel) handleToolResult(msg ToolResultMsg) AppModel {
	m.messages = append(m.messages, DisplayMessage{
		Role:     "tool",
		ToolName: msg.Name,
		Content:  msg.Result,
	})
	m.refreshViewport()
	return m
}

func (m AppModel) handleError(msg ErrorMsg) AppModel {
	m.appendSystemMsg("Error: " + msg.Err.Error())
	m.refreshViewport()
	return m
}

// --- child updates ---

func (m AppModel) updateChildren(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	var tiCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	cmds = append(cmds, tiCmd)

	return m, tea.Batch(cmds...)
}

// --- rendering helpers ---

func (m *AppModel) refreshViewport() {
	content := RenderMessages(m.messages, m.streaming, m.width)
	if m.streaming {
		content += "\n" + m.spinner.View() + " Thinking…"
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *AppModel) appendSystemMsg(text string) {
	m.messages = append(m.messages, DisplayMessage{Role: "system", Content: text})
}

func (m AppModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("5")).
		Render("Ti Code")

	if m.welcome != "" {
		return title + "  " + lipgloss.NewStyle().Faint(true).Render(m.welcome) + "\n" +
			lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", m.width))
	}
	return title + "\n" + lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", m.width))
}

func (m AppModel) renderStatusLine() string {
	sl := StatusLineFromState(m.modelName, m.tokens, m.cost, m.permMode, m.sessionID)
	return lipgloss.NewStyle().Faint(true).Render(sl.Render(m.width))
}
