package tools

import (
	"github.com/settixx/claude-code-go/internal/tools/askuser"
	"github.com/settixx/claude-code-go/internal/tools/bash"
	"github.com/settixx/claude-code-go/internal/tools/brief"
	"github.com/settixx/claude-code-go/internal/tools/configtool"
	"github.com/settixx/claude-code-go/internal/tools/fileedit"
	"github.com/settixx/claude-code-go/internal/tools/fileread"
	"github.com/settixx/claude-code-go/internal/tools/filewrite"
	"github.com/settixx/claude-code-go/internal/tools/glob"
	"github.com/settixx/claude-code-go/internal/tools/grep"
	"github.com/settixx/claude-code-go/internal/tools/notebook"
	"github.com/settixx/claude-code-go/internal/tools/sleep"
	"github.com/settixx/claude-code-go/internal/tools/todo"
	"github.com/settixx/claude-code-go/internal/tools/toolsearch"
	"github.com/settixx/claude-code-go/internal/tools/webfetch"
	"github.com/settixx/claude-code-go/internal/tools/websearch"
	"github.com/settixx/claude-code-go/internal/types"
)

// RegisterCoreTools adds all Tier 1 built-in tools to the registry.
func RegisterCoreTools(registry *types.ToolRegistry) {
	registry.Register(bash.New())
	registry.Register(fileread.New())
	registry.Register(filewrite.New())
	registry.Register(fileedit.New())
	registry.Register(glob.New())
	registry.Register(grep.New())
}

// RegisterExtendedTools adds all Tier 2 extended tools to the registry.
// Must be called after RegisterCoreTools so ToolSearch can discover all tools.
func RegisterExtendedTools(registry *types.ToolRegistry) {
	registry.Register(websearch.New())
	registry.Register(webfetch.New())
	registry.Register(notebook.New())
	registry.Register(todo.New())
	registry.Register(configtool.New())
	registry.Register(sleep.New())
	registry.Register(askuser.New())
	registry.Register(brief.New())
	registry.Register(toolsearch.New(registry))
}
