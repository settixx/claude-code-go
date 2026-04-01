package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/types"
)

// QueryFunc is the callback signature for processing user input.
// It receives the user text and a channel of streaming events to render incrementally.
// Implementations should close the events channel when done.
type QueryFunc func(ctx context.Context, input string, events chan<- types.StreamEvent) error

// AppConfig holds dependencies for the TUI application.
type AppConfig struct {
	// Renderer outputs messages and spinner to the terminal.
	Renderer interfaces.Renderer
	// OnQuery is called for each user input to produce a response.
	OnQuery QueryFunc
	// WelcomeText is displayed once at startup. Empty string skips the banner.
	WelcomeText string
	// Prompt is the input prompt string. Defaults to "> " if empty.
	Prompt string
}

// App is the main TUI application that runs a REPL loop.
type App struct {
	renderer interfaces.Renderer
	input    *InputReader
	onQuery  QueryFunc
	welcome  string
	prompt   string
}

// NewApp creates a TUI application from the given config.
func NewApp(cfg AppConfig) *App {
	prompt := cfg.Prompt
	if prompt == "" {
		prompt = Cyan("> ")
	}
	renderer := cfg.Renderer
	if renderer == nil {
		renderer = NewTermRenderer()
	}
	return &App{
		renderer: renderer,
		input:    NewInputReader(),
		onQuery:  cfg.OnQuery,
		welcome:  cfg.WelcomeText,
		prompt:   prompt,
	}
}

// Run starts the interactive REPL loop. It blocks until the user exits
// (via /exit, Ctrl+D, or Ctrl+C) or the context is cancelled.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.setupSignalHandler(cancel)
	a.printWelcome()

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		line, err := a.input.ReadLine(a.prompt)
		if err == io.EOF {
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, Dim("Goodbye!"))
			return nil
		}
		if err != nil {
			a.renderer.RenderError(err)
			continue
		}

		if line == "" {
			continue
		}

		if handled := a.handleCommand(line); handled {
			if isExitCommand(line) {
				return nil
			}
			continue
		}

		a.processQuery(ctx, line)
	}
}

func (a *App) setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, Dim("Interrupted. Goodbye!"))
		cancel()
	}()
}

func (a *App) printWelcome() {
	if a.welcome == "" {
		return
	}
	fmt.Fprintln(os.Stdout, a.welcome)
	fmt.Fprintln(os.Stdout, HorizontalRule())
	fmt.Fprintln(os.Stdout)
}

func (a *App) processQuery(ctx context.Context, input string) {
	if a.onQuery == nil {
		a.renderer.RenderMessage(types.Message{
			Type: types.MsgSystem,
			Text: "No query handler configured.",
			Level: types.LevelWarning,
		})
		return
	}

	a.renderer.RenderSpinner("Thinking…")

	events := make(chan types.StreamEvent, 64)
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.onQuery(ctx, input, events)
	}()

	a.consumeStream(events)
	a.renderer.StopSpinner()

	if err := <-errCh; err != nil {
		a.renderer.RenderError(err)
	}

	fmt.Fprintln(os.Stdout)
}

func (a *App) consumeStream(events <-chan types.StreamEvent) {
	firstContent := true
	for ev := range events {
		switch ev.Type {
		case types.EventContentBlockStart:
			if firstContent {
				a.renderer.StopSpinner()
				firstContent = false
			}
			if ev.ContentBlock != nil && ev.ContentBlock.Type == types.ContentToolUse {
				fmt.Fprintln(os.Stdout, FormatToolUse(ev.ContentBlock.Name, ""))
			}

		case types.EventContentBlockDelta:
			if firstContent {
				a.renderer.StopSpinner()
				firstContent = false
			}
			if ev.Delta != nil && ev.Delta.Text != "" {
				fmt.Fprint(os.Stdout, ev.Delta.Text)
			}

		case types.EventContentBlockStop:
			fmt.Fprintln(os.Stdout)

		case types.EventError:
			if ev.Error != nil {
				fmt.Fprintln(os.Stdout, Red("Stream error: "+ev.Error.Message))
			}

		case types.EventMessageStop:
			// Final event — nothing more to do.
		}
	}
}

// handleCommand processes slash-commands. Returns true if the line was a command.
func (a *App) handleCommand(line string) bool {
	if !strings.HasPrefix(line, "/") {
		return false
	}

	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/exit", "/quit":
		fmt.Fprintln(os.Stdout, Dim("Goodbye!"))
		return true
	case "/help":
		a.printHelp()
		return true
	case "/clear":
		fmt.Fprint(os.Stdout, "\033[2J\033[H")
		return true
	default:
		fmt.Fprintln(os.Stdout, Yellow("Unknown command: "+cmd+". Type /help for available commands."))
		return true
	}
}

func isExitCommand(line string) bool {
	cmd := strings.ToLower(strings.Fields(line)[0])
	return cmd == "/exit" || cmd == "/quit"
}

func (a *App) printHelp() {
	help := []struct{ cmd, desc string }{
		{"/help", "Show this help message"},
		{"/exit", "Exit the application"},
		{"/quit", "Exit the application"},
		{"/clear", "Clear the screen"},
	}

	fmt.Fprintln(os.Stdout, Bold("Available commands:"))
	for _, h := range help {
		fmt.Fprintf(os.Stdout, "  %-12s %s\n", Cyan(h.cmd), h.desc)
	}
	fmt.Fprintln(os.Stdout)
}
