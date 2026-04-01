package tools

import (
	"context"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/skills"
	"github.com/settixx/claude-code-go/internal/tools/agent"
	"github.com/settixx/claude-code-go/internal/tools/askuser"
	"github.com/settixx/claude-code-go/internal/tools/bash"
	"github.com/settixx/claude-code-go/internal/tools/brief"
	"github.com/settixx/claude-code-go/internal/tools/configtool"
	"github.com/settixx/claude-code-go/internal/tools/fileedit"
	"github.com/settixx/claude-code-go/internal/tools/fileread"
	"github.com/settixx/claude-code-go/internal/tools/filewrite"
	"github.com/settixx/claude-code-go/internal/tools/glob"
	"github.com/settixx/claude-code-go/internal/tools/grep"
	"github.com/settixx/claude-code-go/internal/tools/listmcpresources"
	"github.com/settixx/claude-code-go/internal/tools/mcptool"
	"github.com/settixx/claude-code-go/internal/tools/notebook"
	"github.com/settixx/claude-code-go/internal/tools/planmode"
	"github.com/settixx/claude-code-go/internal/tools/readmcpresource"
	"github.com/settixx/claude-code-go/internal/tools/sleep"
	"github.com/settixx/claude-code-go/internal/tools/tasktool"
	"github.com/settixx/claude-code-go/internal/tools/teamtool"
	"github.com/settixx/claude-code-go/internal/tools/todo"
	"github.com/settixx/claude-code-go/internal/tools/toolsearch"
	"github.com/settixx/claude-code-go/internal/tools/webfetch"
	"github.com/settixx/claude-code-go/internal/tools/websearch"
	"github.com/settixx/claude-code-go/internal/tools/worktree"
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

// MCPProvider unifies the three MCP tool interfaces into one dependency.
// The mcp.ManagerAdapter satisfies this interface.
type MCPProvider interface {
	CallTool(ctx context.Context, server, tool string, args map[string]interface{}) (string, error)
	ListResources(ctx context.Context, server string) ([]types.ServerResource, error)
	ConnectedServers() []string
	ReadResource(ctx context.Context, server, uri string) (string, error)
}

// AdvancedToolDeps bundles the external dependencies required by Tier 3 tools.
type AdvancedToolDeps struct {
	WorkerPool    *coordinator.WorkerPool
	EngineFactory agent.EngineFactory
	TaskStore     *tasktool.TaskStore
	StateStore    interfaces.StateStore
	MCPManager    MCPProvider
	SkillRegistry *skills.SkillRegistry
	TeamManager   *coordinator.TeamManager
	CWD           string
}

// RegisterAdvancedTools adds all Tier 3 tools that require runtime dependencies
// (coordinator, task store, MCP connections, etc.) to the registry.
func RegisterAdvancedTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	registerAgentTools(registry, deps)
	registerTaskTools(registry, deps)
	registerPlanModeTools(registry, deps)
	registerMCPTools(registry, deps)
	registerWorktreeTools(registry, deps)
	registerTeamTools(registry, deps)
	registerSkillTool(registry, deps)
}

func registerAgentTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	if deps.WorkerPool == nil {
		return
	}
	registry.Register(agent.NewTool(deps.WorkerPool, deps.EngineFactory, deps.CWD))
	registry.Register(agent.NewSendMessageTool(deps.WorkerPool))
}

func registerTaskTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	if deps.TaskStore == nil {
		return
	}
	registry.Register(tasktool.NewCreateTool(deps.TaskStore))
	registry.Register(tasktool.NewListTool(deps.TaskStore))
	registry.Register(tasktool.NewGetTool(deps.TaskStore))
	registry.Register(tasktool.NewUpdateTool(deps.TaskStore))
	registry.Register(tasktool.NewStopTool(deps.TaskStore))
	registry.Register(tasktool.NewOutputTool(deps.TaskStore))
}

func registerPlanModeTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	if deps.StateStore == nil {
		return
	}
	registry.Register(planmode.NewEnterTool(deps.StateStore))
	registry.Register(planmode.NewExitTool(deps.StateStore))
}

func registerMCPTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	if deps.MCPManager == nil {
		return
	}
	registry.Register(mcptool.New(deps.MCPManager))
	registry.Register(listmcpresources.New(deps.MCPManager))
	registry.Register(readmcpresource.New(deps.MCPManager))
}

func registerWorktreeTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	registry.Register(worktree.NewEnterTool(deps.CWD))
	registry.Register(worktree.NewExitTool())
}

func registerTeamTools(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	if deps.TeamManager == nil {
		return
	}
	registry.Register(teamtool.NewCreateTool(deps.TeamManager))
	registry.Register(teamtool.NewDeleteTool(deps.TeamManager))
}

func registerSkillTool(registry *types.ToolRegistry, deps AdvancedToolDeps) {
	if deps.SkillRegistry == nil {
		return
	}
	registry.Register(skills.NewSkillTool(deps.SkillRegistry))
}
