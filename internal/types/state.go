package types

// TaskStatus describes where a task is in its lifecycle.
type TaskStatus string

const (
	TaskPending  TaskStatus = "pending"
	TaskRunning  TaskStatus = "running"
	TaskStopped  TaskStatus = "stopped"
	TaskComplete TaskStatus = "complete"
	TaskFailed   TaskStatus = "failed"
)

// TaskKind distinguishes different task execution models.
type TaskKind string

const (
	TaskLocalShell     TaskKind = "local_shell"
	TaskLocalAgent     TaskKind = "local_agent"
	TaskRemoteAgent    TaskKind = "remote_agent"
	TaskInProcessAgent TaskKind = "in_process_agent"
	TaskLocalWorkflow  TaskKind = "local_workflow"
	TaskMonitorMCP     TaskKind = "monitor_mcp"
	TaskDream          TaskKind = "dream"
)

// TaskState describes a single tracked task.
type TaskState struct {
	ID             string     `json:"id"`
	Kind           TaskKind   `json:"kind"`
	Status         TaskStatus `json:"status"`
	Name           string     `json:"name,omitempty"`
	AgentID        AgentId    `json:"agent_id,omitempty"`
	IsBackgrounded bool       `json:"is_backgrounded"`
	WorktreePath   string     `json:"worktree_path,omitempty"`
}

// IsBackgroundTask reports whether a task should appear in the background indicator.
func (t *TaskState) IsBackgroundTask() bool {
	if t.Status != TaskRunning && t.Status != TaskPending {
		return false
	}
	return t.IsBackgrounded
}

// FooterItem identifies which footer pill is focused.
type FooterItem string

const (
	FooterTasks     FooterItem = "tasks"
	FooterTmux      FooterItem = "tmux"
	FooterBagel     FooterItem = "bagel"
	FooterTeams     FooterItem = "teams"
	FooterBridge    FooterItem = "bridge"
	FooterCompanion FooterItem = "companion"
)

// AppState is the centralized reactive state for the entire application.
type AppState struct {
	Settings            Settings                 `json:"settings"`
	Verbose             bool                     `json:"verbose"`
	MainLoopModel       string                   `json:"main_loop_model"`
	StatusLineText      string                   `json:"status_line_text,omitempty"`
	IsBriefOnly         bool                     `json:"is_brief_only"`
	ThinkingEnabled     *bool                    `json:"thinking_enabled,omitempty"`
	PermissionMode      PermissionMode           `json:"permission_mode"`
	Agent               string                   `json:"agent,omitempty"`
	FooterSelection     FooterItem               `json:"footer_selection,omitempty"`

	Tasks               map[string]*TaskState    `json:"tasks"`
	AgentNameRegistry   map[string]AgentId       `json:"agent_name_registry"`
	ForegroundedTaskID  string                   `json:"foregrounded_task_id,omitempty"`
	ViewingAgentTaskID  string                   `json:"viewing_agent_task_id,omitempty"`

	CompanionReaction   string                   `json:"companion_reaction,omitempty"`
	CompanionPetAt      int64                    `json:"companion_pet_at,omitempty"`

	MCP                 MCPState                 `json:"mcp"`
}

// MCPState tracks connected MCP servers, tools, and resources.
type MCPState struct {
	Clients            []MCPServerConnection     `json:"clients"`
	Tools              []Tool                    `json:"-"`
	Resources          map[string][]ServerResource `json:"resources"`
	PluginReconnectKey int                       `json:"plugin_reconnect_key"`
}

// MCPServerConnection represents one connected MCP server.
type MCPServerConnection struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ServerResource represents a single MCP resource.
type ServerResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}
