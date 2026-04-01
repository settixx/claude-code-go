package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/api"
	"github.com/settixx/claude-code-go/internal/config"
	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/permissions"
	"github.com/settixx/claude-code-go/internal/query"
	"github.com/settixx/claude-code-go/internal/state"
	"github.com/settixx/claude-code-go/internal/storage"
	"github.com/settixx/claude-code-go/internal/tools"
	"github.com/settixx/claude-code-go/internal/types"
)

// PrintConfig holds parameters for non-interactive (print) mode.
type PrintConfig struct {
	Model        string
	Verbose      bool
	OutputFormat string
}

// RunPrint executes a single prompt in non-interactive mode,
// streams the response to stdout, and returns.
func RunPrint(ctx context.Context, prompt string, cfg PrintConfig) error {
	if prompt == "" {
		return fmt.Errorf("no prompt provided for print mode")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	model := cfg.Model
	if model == "" {
		model = resolveModel("")
	}

	apiKey := resolveAPIKey(CLIFlags{}, cwd)
	if apiKey == "" {
		return fmt.Errorf("no API key found. Set ANTHROPIC_API_KEY env var, or add customApiKey to ~/.claude/settings.json")
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[print] model=%s format=%s prompt_len=%d\n",
			model, outputFormat(cfg.OutputFormat), len(prompt))
	}

	llmClient := api.NewClient(api.ClientConfig{
		APIKey:       apiKey,
		DefaultModel: model,
	})

	toolRegistry := types.NewToolRegistry()
	tools.RegisterCoreTools(toolRegistry)
	tools.RegisterExtendedTools(toolRegistry)

	checker := permissions.NewChecker(types.PermBypassPermissions, nil)
	executor := NewPermissionAwareExecutor(toolRegistry, checker)

	sessionStore := storage.NewFileStorage(config.SessionDir())

	stateStore := state.NewStore(types.AppState{
		MainLoopModel:     model,
		Verbose:           cfg.Verbose,
		PermissionMode:    types.PermBypassPermissions,
		Tasks:             make(map[string]*types.TaskState),
		AgentNameRegistry: make(map[string]types.AgentId),
	})

	renderer, jsonCollector := selectPrintRenderer(cfg.OutputFormat)

	engine := query.NewEngine(query.EngineConfig{
		LLMClient:      llmClient,
		ToolExecutor:   executor,
		StateStore:     stateStore,
		SessionStorage: sessionStore,
		Renderer:       renderer,
		Model:          model,
		CWD:            cwd,
	})

	if err := engine.Run(ctx, prompt); err != nil {
		slog.Error("print mode engine run failed", "error", err)
		return err
	}

	if jsonCollector != nil {
		return jsonCollector.Flush()
	}
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

// selectPrintRenderer picks the appropriate renderer for print mode based on
// the output format. Returns the renderer and, for "json" format only, the
// collector that must be flushed after the engine finishes.
func selectPrintRenderer(format string) (interfaces.Renderer, *JSONCollectRenderer) {
	switch format {
	case "stream-json":
		return NewJSONStreamRenderer(), nil
	case "json":
		c := NewJSONCollectRenderer()
		return c, c
	default:
		return NewStdRenderer(), nil
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
