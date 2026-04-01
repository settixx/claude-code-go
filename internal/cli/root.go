package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/api"
	"github.com/settixx/claude-code-go/internal/config"
	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/mcp"
	"github.com/settixx/claude-code-go/internal/permissions"
	"github.com/settixx/claude-code-go/internal/query"
	"github.com/settixx/claude-code-go/internal/state"
	"github.com/settixx/claude-code-go/internal/storage"
	"github.com/settixx/claude-code-go/internal/tools"
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
	OutputFormat   string
	PermissionMode string
	Resume         string
	Continue       bool
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
			Model:        flags.Model,
			Verbose:      flags.Verbose,
			OutputFormat: flags.OutputFormat,
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
	flag.StringVar(&f.OutputFormat, "output-format", "text", "Output format: text, json, stream-json")
	flag.StringVar(&f.OutputFormat, "o", "text", "Output format (shorthand)")
	flag.StringVar(&f.PermissionMode, "permission-mode", "default", "Set permission mode")
	flag.StringVar(&f.Resume, "resume", "", "Resume a previous session by ID")
	flag.StringVar(&f.Resume, "r", "", "Resume a session (shorthand)")
	flag.BoolVar(&f.Continue, "continue", false, "Continue from the most recent session")
	flag.BoolVar(&f.Continue, "c", false, "Continue session (shorthand)")
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
	fmt.Fprintf(os.Stderr, "  ti-code --continue <session-id>  Continue a session\n")
}

func runPrintMode(ctx context.Context, flags CLIFlags, args []string) error {
	prompt := strings.Join(args, " ")
	stdinPrompt := ReadStdinPrompt()
	merged := MergePromptWithStdin(prompt, stdinPrompt)
	if merged == "" {
		return fmt.Errorf("print mode requires a prompt (positional arg or stdin pipe)")
	}
	return RunPrint(ctx, merged, PrintConfig{
		Model:        flags.Model,
		Verbose:      flags.Verbose,
		OutputFormat: flags.OutputFormat,
	})
}

// runInteractive wires every module together and starts the TUI REPL.
func runInteractive(ctx context.Context, flags CLIFlags) error {
	registry := NewCommandRegistry()
	RegisterDefaultCommands(registry)
	setDefaultRegistry(registry)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	configProvider, err := config.NewProvider(cwd)
	if err != nil {
		slog.Debug("config load failed", "error", err)
	}

	model := resolveModel(flags.Model)
	apiKey := resolveAPIKeyFromProvider(configProvider)
	if apiKey == "" {
		return fmt.Errorf("no API key found. Set ANTHROPIC_API_KEY env var, or add customApiKey to ~/.claude/settings.json")
	}

	permMode := parsePermissionMode(flags.PermissionMode)

	var ruleSet *permissions.RuleSet
	if configProvider != nil {
		if claudeRules := configProvider.GetClaudeMDRules(); claudeRules != nil && claudeRules.Content != "" {
			ruleSet, _ = permissions.ParsePermissionRules(claudeRules.Content)
		}

		pc := configProvider.GetProjectConfig()
		if ruleSet == nil {
			ruleSet = permissions.NewRuleSet()
		}
		for _, pat := range pc.AllowedTools {
			ruleSet.AddAllowRule(pat)
		}
	}

	llmClient := api.NewClient(api.ClientConfig{
		APIKey:       apiKey,
		DefaultModel: model,
	})

	toolRegistry := types.NewToolRegistry()
	tools.RegisterCoreTools(toolRegistry)
	tools.RegisterExtendedTools(toolRegistry)

	// MCP auto-startup: load configs, connect servers, register tools,
	// and produce instruction text for the system prompt.
	mcpAppend := bootstrapMCP(ctx, configProvider, toolRegistry)

	checker := permissions.NewChecker(permMode, ruleSet)
	executor := NewPermissionAwareExecutor(toolRegistry, checker)

	sessionStore := storage.NewFileStorage(config.SessionDir())

	var initialMessages []types.Message

	resumeID := resolveSessionToResume(flags, sessionStore)
	if resumeID != "" {
		msgs, loadErr := sessionStore.Load(types.SessionId(resumeID))
		if loadErr != nil {
			slog.Warn("failed to load session for continue/resume", "id", resumeID, "error", loadErr)
		} else if len(msgs) > 0 {
			initialMessages = msgs
			fmt.Fprintf(os.Stdout, "%s Continuing session %s (%d messages)\n",
				tui.Blue("ℹ"), tui.Cyan(resumeID), len(msgs))
		}
	}

	stateStore := state.NewStore(types.AppState{
		MainLoopModel:     model,
		Verbose:           flags.Verbose,
		PermissionMode:    permMode,
		Agent:             flags.Agent,
		Tasks:             make(map[string]*types.TaskState),
		AgentNameRegistry: make(map[string]types.AgentId),
	})

	renderer := NewTUIRenderer()

	engine := query.NewEngine(query.EngineConfig{
		LLMClient:       llmClient,
		ToolExecutor:    executor,
		StateStore:      stateStore,
		SessionStorage:  sessionStore,
		Renderer:        renderer,
		Model:           model,
		CWD:             cwd,
		AppendPrompt:    mcpAppend,
		InitialMessages: initialMessages,
	})

	realHandler := buildEngineQueryHandler(engine, renderer)

	cmdCtx := &CommandContext{
		Model:             model,
		Verbose:           flags.Verbose,
		PermissionMode:    flags.PermissionMode,
		CWD:               cwd,
		Storage:           sessionStore,
		StateStore:        stateStore,
		PermissionChecker: checker,
	}
	queryHandler := buildCommandInterceptor(registry, cmdCtx, realHandler)

	welcomeText := buildWelcome(model)

	app := tui.NewApp(tui.AppConfig{
		WelcomeText: welcomeText,
		OnQuery:     queryHandler,
	})

	tuiPrompter := NewTUIPrompter(app)
	checker.SetPrompter(tuiPrompter)

	return app.Run(ctx)
}

// bootstrapMCP loads MCP server configs from user + project settings,
// connects to all servers, registers their tools into the registry,
// and returns formatted instruction text for the system prompt.
func bootstrapMCP(ctx context.Context, provider *config.Provider, registry *types.ToolRegistry) string {
	if provider == nil {
		return ""
	}

	settings := provider.GetSettings()
	servers := mergeServerConfigs(settings.User.McpServers, settings.Project.McpServers)
	if len(servers) == 0 {
		return ""
	}

	result, err := mcp.Bootstrap(ctx, servers)
	if err != nil {
		slog.Warn("mcp bootstrap failed", "error", err)
		return ""
	}

	for _, t := range result.Tools {
		registry.Register(t)
	}

	return result.Manager.FormatInstructionsText()
}

// mergeServerConfigs combines user-level and project-level MCP server
// configs. Project entries override user entries with the same name.
func mergeServerConfigs(user, project map[string]types.McpServerConfig) map[string]types.McpServerConfig {
	merged := make(map[string]types.McpServerConfig, len(user)+len(project))
	for k, v := range user {
		merged[k] = v
	}
	for k, v := range project {
		merged[k] = v
	}
	return merged
}

// buildEngineQueryHandler creates a tui.QueryFunc that bridges the
// query.Engine into the TUI's streaming event channel.
func buildEngineQueryHandler(engine *query.Engine, renderer *TUIRenderer) tui.QueryFunc {
	return func(ctx context.Context, input string, events chan<- types.StreamEvent) error {
		renderer.SetEvents(events)
		defer func() {
			renderer.ClearEvents()
			close(events)
		}()
		return engine.Run(ctx, input)
	}
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

// resolveAPIKeyFromProvider resolves the API key from (in priority order):
// 1. Environment variable ANTHROPIC_API_KEY
// 2. User config via the already-created Provider
func resolveAPIKeyFromProvider(provider *config.Provider) string {
	if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
		return envKey
	}
	if provider == nil {
		return ""
	}
	if key := provider.GetSettings().User.CustomApiKey; key != "" {
		return key
	}
	return ""
}

// resolveAPIKey resolves the API key from (in priority order):
// 1. Environment variable ANTHROPIC_API_KEY
// 2. User config file (~/.claude/settings.json -> customApiKey)
func resolveAPIKey(flags CLIFlags, cwd string) string {
	if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
		return envKey
	}

	cfgProvider, err := config.NewProvider(cwd)
	if err != nil {
		slog.Debug("config load failed during API key resolution", "error", err)
		return ""
	}
	if key := cfgProvider.GetSettings().User.CustomApiKey; key != "" {
		return key
	}
	return ""
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

func parsePermissionMode(s string) types.PermissionMode {
	switch strings.ToLower(s) {
	case "plan":
		return types.PermPlan
	case "acceptedits":
		return types.PermAcceptEdits
	case "bypasspermissions":
		return types.PermBypassPermissions
	case "dontask":
		return types.PermDontAsk
	case "auto":
		return types.PermAuto
	default:
		return types.PermDefault
	}
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

// coalesce returns the first non-empty string.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// resolveSessionToResume picks the session ID to load based on CLI flags.
// --resume <id> takes an explicit ID; --continue loads the most recent session.
func resolveSessionToResume(flags CLIFlags, store interfaces.SessionStorage) string {
	if flags.Resume != "" {
		return flags.Resume
	}
	if !flags.Continue {
		return ""
	}

	sessions, err := store.List()
	if err != nil || len(sessions) == 0 {
		slog.Warn("--continue: no previous sessions found")
		return ""
	}
	return string(sessions[0].ID)
}
