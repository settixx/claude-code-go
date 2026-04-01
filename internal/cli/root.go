package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/tui"
	"github.com/settixx/claude-code-go/internal/types"
	"github.com/settixx/claude-code-go/internal/version"
)

// CLIFlags holds all parsed command-line flags.
type CLIFlags struct {
	Version        bool
	Model          string
	Verbose        bool
	Print          bool
	PermissionMode string
	Resume         string
	Agent          string
	Debug          bool
}

// Execute is the main entry point for the Ti Code CLI.
// It parses flags, dispatches to print mode or the interactive REPL,
// and returns any fatal error.
func Execute(ctx context.Context) error {
	flags := parseFlags()

	if flags.Version {
		fmt.Fprintln(os.Stdout, version.Full())
		return nil
	}

	positionalArgs := flag.Args()

	if flags.Print {
		return runPrintMode(ctx, flags, positionalArgs)
	}

	stdinPrompt := ReadStdinPrompt()
	if stdinPrompt != "" {
		argPrompt := strings.Join(positionalArgs, " ")
		merged := MergePromptWithStdin(argPrompt, stdinPrompt)
		return RunPrint(ctx, merged, PrintConfig{
			Model:   flags.Model,
			Verbose: flags.Verbose,
		})
	}

	return runInteractive(ctx, flags)
}

func parseFlags() CLIFlags {
	var f CLIFlags

	flag.BoolVar(&f.Version, "version", false, "Print version and exit")
	flag.StringVar(&f.Model, "model", "", "Model selection")
	flag.StringVar(&f.Model, "m", "", "Model selection (shorthand)")
	flag.BoolVar(&f.Verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&f.Verbose, "v", false, "Enable verbose output (shorthand)")
	flag.BoolVar(&f.Print, "print", false, "Non-interactive mode: send prompt, print response, exit")
	flag.BoolVar(&f.Print, "p", false, "Non-interactive mode (shorthand)")
	flag.StringVar(&f.PermissionMode, "permission-mode", "default", "Set permission mode")
	flag.StringVar(&f.Resume, "resume", "", "Resume a previous session by ID")
	flag.StringVar(&f.Resume, "r", "", "Resume a session (shorthand)")
	flag.StringVar(&f.Agent, "agent", "", "Select agent")
	flag.BoolVar(&f.Debug, "debug", false, "Enable debug output")

	flag.Usage = printUsage
	flag.Parse()

	return f
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: ti-code [flags] [prompt...]\n\n")
	fmt.Fprintf(os.Stderr, "Ti Code — an AI coding assistant in your terminal\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  ti-code                          Start interactive REPL\n")
	fmt.Fprintf(os.Stderr, "  ti-code -p \"explain this code\"   Print mode\n")
	fmt.Fprintf(os.Stderr, "  cat file.go | ti-code -p         Pipe input in print mode\n")
	fmt.Fprintf(os.Stderr, "  ti-code --resume <session-id>    Resume a session\n")
}

func runPrintMode(ctx context.Context, flags CLIFlags, args []string) error {
	prompt := strings.Join(args, " ")
	stdinPrompt := ReadStdinPrompt()
	merged := MergePromptWithStdin(prompt, stdinPrompt)
	if merged == "" {
		return fmt.Errorf("print mode requires a prompt (positional arg or stdin pipe)")
	}
	return RunPrint(ctx, merged, PrintConfig{
		Model:   flags.Model,
		Verbose: flags.Verbose,
	})
}

func runInteractive(ctx context.Context, flags CLIFlags) error {
	registry := NewCommandRegistry()
	RegisterDefaultCommands(registry)
	setDefaultRegistry(registry)

	model := resolveModel(flags.Model)

	cmdCtx := &CommandContext{
		Model:          model,
		Verbose:        flags.Verbose,
		PermissionMode: flags.PermissionMode,
	}

	welcomeText := buildWelcome(model)

	echoHandler := buildPrintStreamHandler()
	queryHandler := buildCommandInterceptor(registry, cmdCtx, echoHandler)

	app := tui.NewApp(tui.AppConfig{
		WelcomeText: welcomeText,
		OnQuery:     queryHandler,
	})

	if flags.Resume != "" {
		fmt.Fprintf(os.Stdout, "%s Resuming session %s… (not yet implemented)\n",
			tui.Blue("ℹ"), tui.Cyan(flags.Resume))
	}

	return app.Run(ctx)
}

// buildCommandInterceptor wraps a real query handler so that slash commands
// are routed through the CommandRegistry first. Non-command input falls
// through to the inner handler.
func buildCommandInterceptor(reg *CommandRegistry, cmdCtx *CommandContext, inner tui.QueryFunc) tui.QueryFunc {
	return func(ctx context.Context, input string, events chan<- types.StreamEvent) error {
		if !strings.HasPrefix(input, "/") {
			return inner(ctx, input, events)
		}

		defer close(events)

		handled, err := reg.Execute(input, cmdCtx)
		if !handled {
			return inner(ctx, input, events)
		}
		if err == ErrExit {
			return ErrExit
		}
		if err != nil {
			events <- types.StreamEvent{
				Type: types.EventError,
				Error: &struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{Type: "command_error", Message: err.Error()},
			}
		}
		return nil
	}
}

func resolveModel(flagModel string) string {
	if flagModel != "" {
		return flagModel
	}
	if env := os.Getenv("TI_CODE_MODEL"); env != "" {
		return env
	}
	return "claude-sonnet-4-20250514"
}

func buildWelcome(model string) string {
	var b strings.Builder
	b.WriteString(tui.Bold("Ti Code") + " " + tui.Dim(version.Short()))
	b.WriteString("\n")
	b.WriteString(tui.Dim("Model: " + model))
	b.WriteString("\n")
	b.WriteString(tui.Dim("Type /help for commands, /exit to quit"))
	return b.String()
}
