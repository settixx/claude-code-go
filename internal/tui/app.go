package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/settixx/claude-code-go/internal/types"
)

// QueryFunc is the callback signature for processing user input.
// It receives the user text and a channel of streaming events to render incrementally.
// Implementations should close the events channel when done.
type QueryFunc func(ctx context.Context, input string, events chan<- types.StreamEvent) error

// AppConfig holds dependencies for the TUI application.
type AppConfig struct {
	// OnQuery is called for each user input to produce a response.
	OnQuery QueryFunc
	// WelcomeText is displayed once at startup. Empty string skips the banner.
	WelcomeText string
	// Prompt mode: "normal" or "plan".
	PromptMode string
}

// App is the main TUI application backed by Bubble Tea.
type App struct {
	cfg     AppConfig
	program *tea.Program
}

// NewApp creates a TUI application from the given config.
func NewApp(cfg AppConfig) *App {
	return &App{cfg: cfg}
}

// Run starts the interactive Bubble Tea TUI. It blocks until the user exits.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := NewAppModel(a.cfg.WelcomeText, func(input string) {
		go a.processQuery(ctx, input)
	})

	if a.cfg.PromptMode != "" {
		model.textInput.SetMode(a.cfg.PromptMode)
	}

	a.program = tea.NewProgram(model, tea.WithAltScreen())

	if _, err := a.program.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}

// Send injects a Bubble Tea message into the running program from the outside.
// Safe to call from any goroutine.
func (a *App) Send(msg tea.Msg) {
	if a.program != nil {
		a.program.Send(msg)
	}
}

// processQuery runs the OnQuery callback and bridges streaming events
// into Bubble Tea messages via program.Send.
func (a *App) processQuery(ctx context.Context, input string) {
	if a.cfg.OnQuery == nil {
		a.Send(ErrorMsg{Err: fmt.Errorf("no query handler configured")})
		return
	}

	events := make(chan types.StreamEvent, 64)
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.cfg.OnQuery(ctx, input, events)
	}()

	a.consumeStream(events)

	if err := <-errCh; err != nil {
		a.Send(ErrorMsg{Err: err})
	}

	a.Send(StreamDoneMsg{})
}

// consumeStream reads API streaming events and translates them to TUI messages.
func (a *App) consumeStream(events <-chan types.StreamEvent) {
	for ev := range events {
		switch ev.Type {
		case types.EventMessageStart:
			if ev.Message != nil && ev.Message.Usage != nil {
				a.Send(TokenUsageMsg{
					InputTokens:  ev.Message.Usage.InputTokens,
					OutputTokens: ev.Message.Usage.OutputTokens,
				})
			}

		case types.EventContentBlockStart:
			if ev.ContentBlock != nil && ev.ContentBlock.Type == types.ContentToolUse {
				a.Send(ToolCallMsg{Name: ev.ContentBlock.Name})
			}

		case types.EventContentBlockDelta:
			if ev.Delta != nil && ev.Delta.Text != "" {
				a.Send(StreamChunkMsg{Text: ev.Delta.Text})
			}

		case types.EventContentBlockStop:
			// block boundary — nothing extra needed

		case types.EventError:
			if ev.Error != nil {
				a.Send(ErrorMsg{Err: fmt.Errorf(ev.Error.Message)})
			}

		case types.EventMessageStop:
			if ev.Usage != nil {
				a.Send(TokenUsageMsg{
					InputTokens:  ev.Usage.InputTokens,
					OutputTokens: ev.Usage.OutputTokens,
				})
			}
		}
	}
}

// SendPermissionRequest presents a permission dialog and blocks until the user responds.
func (a *App) SendPermissionRequest(tool, input string) bool {
	ch := make(chan bool, 1)
	a.Send(PermissionRequestMsg{Tool: tool, Input: input, ResponseCh: ch})
	return <-ch
}

// PrintError is a convenience for writing an error outside the TUI lifecycle.
func PrintError(err error) {
	fmt.Fprintln(os.Stderr, Red("✗ Error: "+err.Error()))
}
