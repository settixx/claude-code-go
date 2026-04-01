package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/tui"
	"github.com/settixx/claude-code-go/internal/types"
)

// PrintConfig holds parameters for non-interactive (print) mode.
type PrintConfig struct {
	// Model overrides the default model for this invocation.
	Model string
	// Verbose enables extra diagnostic output.
	Verbose bool
	// OutputFormat selects the output format: "text" (default), "json", or "stream-json".
	OutputFormat string
}

// RunPrint executes a single prompt in non-interactive mode,
// streams the response to stdout, and returns.
func RunPrint(ctx context.Context, prompt string, cfg PrintConfig) error {
	if prompt == "" {
		return fmt.Errorf("no prompt provided for print mode")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[print] model=%s format=%s prompt_len=%d\n",
			model, outputFormat(cfg.OutputFormat), len(prompt))
	}

	fmt.Fprintln(os.Stdout, tui.Dim("(print mode — LLM client not yet connected)"))
	fmt.Fprintf(os.Stdout, "Prompt: %s\n", prompt)
	fmt.Fprintf(os.Stdout, "Model:  %s\n", model)
	fmt.Fprintln(os.Stdout, tui.Blue("ℹ Response streaming will be available once LLMClient is wired."))

	return nil
}

// ReadStdinPrompt reads all of stdin (for piped input) and returns it as a prompt string.
// Returns empty string if stdin is a terminal.
func ReadStdinPrompt() string {
	if isTerminal(os.Stdin) {
		return ""
	}

	var b strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		b.WriteString(scanner.Text())
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return true
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func outputFormat(f string) string {
	if f == "" {
		return "text"
	}
	return f
}

// buildPrintStreamHandler returns a tui.QueryFunc stub for print mode
// that writes streaming events to stdout. This is a temporary shim until
// the real LLMClient is integrated.
func buildPrintStreamHandler() tui.QueryFunc {
	return func(ctx context.Context, input string, events chan<- types.StreamEvent) error {
		defer close(events)

		events <- types.StreamEvent{
			Type: types.EventContentBlockStart,
			ContentBlock: &types.ContentBlock{
				Type: types.ContentText,
			},
		}
		events <- types.StreamEvent{
			Type: types.EventContentBlockDelta,
			Delta: &types.DeltaBlock{
				Type: "text_delta",
				Text: "(LLM client not yet connected — echo) " + input,
			},
		}
		events <- types.StreamEvent{Type: types.EventContentBlockStop}
		events <- types.StreamEvent{Type: types.EventMessageStop}
		return nil
	}
}

// MergePromptWithStdin combines a CLI positional arg with piped stdin.
// If both are present, stdin is appended after a blank line.
func MergePromptWithStdin(argPrompt, stdinPrompt string) string {
	argPrompt = strings.TrimSpace(argPrompt)
	stdinPrompt = strings.TrimSpace(stdinPrompt)

	switch {
	case argPrompt != "" && stdinPrompt != "":
		return argPrompt + "\n\n" + stdinPrompt
	case stdinPrompt != "":
		return stdinPrompt
	default:
		return argPrompt
	}
}

// StreamToWriter copies streaming events to a writer (for non-interactive output).
func StreamToWriter(w io.Writer, events <-chan types.StreamEvent) {
	for ev := range events {
		if ev.Type != types.EventContentBlockDelta {
			continue
		}
		if ev.Delta == nil || ev.Delta.Text == "" {
			continue
		}
		fmt.Fprint(w, ev.Delta.Text)
	}
	fmt.Fprintln(w)
}
