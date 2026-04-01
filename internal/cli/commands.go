package cli

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// CommandHandler is the function signature for slash command handlers.
// It receives the raw arguments string (everything after the command name)
// and a CommandContext carrying runtime dependencies.
type CommandHandler func(args string, ctx *CommandContext) error

// Command describes a single slash command.
type Command struct {
	Name        string
	Description string
	Aliases     []string
	Handler     CommandHandler
}

// CommandContext carries runtime state available to every command handler.
type CommandContext struct {
	// Model is the currently active LLM model name.
	Model string
	// Verbose indicates whether verbose output is enabled.
	Verbose bool
	// PermissionMode is the active permission mode string.
	PermissionMode string
	// SessionID is the current session identifier (empty if none).
	SessionID string

	// TokensIn tracks cumulative input tokens for the session.
	TokensIn int
	// TokensOut tracks cumulative output tokens for the session.
	TokensOut int
	// CostUSD tracks cumulative estimated cost in USD.
	CostUSD float64
}

// CommandRegistry holds all registered slash commands and provides lookup.
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]*Command
	aliases  map[string]string
}

// NewCommandRegistry creates an empty registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
		aliases:  make(map[string]string),
	}
}

// Register adds a command to the registry. The command name is stored
// without the leading slash.
func (r *CommandRegistry) Register(cmd *Command) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.TrimPrefix(cmd.Name, "/")
	r.commands[name] = cmd
	for _, alias := range cmd.Aliases {
		r.aliases[strings.TrimPrefix(alias, "/")] = name
	}
}

// Find looks up a command by name or alias.
// The input should NOT include the leading slash.
func (r *CommandRegistry) Find(name string) *Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name = strings.TrimPrefix(name, "/")
	if cmd, ok := r.commands[name]; ok {
		return cmd
	}
	if canonical, ok := r.aliases[name]; ok {
		return r.commands[canonical]
	}
	return nil
}

// All returns every registered command sorted by name.
func (r *CommandRegistry) All() []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmds := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
	return cmds
}

// Execute parses a slash-command line and runs the matching handler.
// Returns true if the input was recognized as a command.
func (r *CommandRegistry) Execute(line string, ctx *CommandContext) (handled bool, err error) {
	if !strings.HasPrefix(line, "/") {
		return false, nil
	}

	parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
	name := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	cmd := r.Find(name)
	if cmd == nil {
		return true, fmt.Errorf("unknown command: /%s — type /help for available commands", name)
	}
	return true, cmd.Handler(args, ctx)
}

// RegisterDefaultCommands populates the registry with all built-in slash commands.
func RegisterDefaultCommands(reg *CommandRegistry) {
	registerCoreCommands(reg)
	registerSessionCommands(reg)
	registerDevCommands(reg)
	registerAdvancedCommands(reg)
}
