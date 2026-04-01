package types

// PermissionMode controls how permission checks behave.
type PermissionMode string

const (
	PermDefault          PermissionMode = "default"
	PermPlan             PermissionMode = "plan"
	PermAcceptEdits      PermissionMode = "acceptEdits"
	PermBypassPermissions PermissionMode = "bypassPermissions"
	PermDontAsk          PermissionMode = "dontAsk"
	PermAuto             PermissionMode = "auto"
	PermBubble           PermissionMode = "bubble"
)

// PermissionBehavior is the outcome for a single tool check.
type PermissionBehavior string

const (
	BehaviorAllow PermissionBehavior = "allow"
	BehaviorDeny  PermissionBehavior = "deny"
	BehaviorAsk   PermissionBehavior = "ask"
)

// McpServerConfig describes one MCP server entry in settings.
type McpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// ProjectConfig holds per-project settings from .claude/settings.json.
type ProjectConfig struct {
	AllowedTools    []string                    `json:"allowedTools,omitempty"`
	McpContextURIs  []string                    `json:"mcpContextUris,omitempty"`
	McpServers      map[string]McpServerConfig  `json:"mcpServers,omitempty"`
	HasTrustDialogAccepted bool                 `json:"hasTrustDialogAccepted,omitempty"`
}

// UserConfig holds user-global settings from ~/.claude/settings.json.
type UserConfig struct {
	CustomApiKey   string                     `json:"customApiKey,omitempty"`
	DefaultModel   string                     `json:"defaultModel,omitempty"`
	DefaultMode    PermissionMode             `json:"defaultMode,omitempty"`
	Theme          string                     `json:"theme,omitempty"`
	Verbose        bool                       `json:"verbose,omitempty"`
	McpServers     map[string]McpServerConfig `json:"mcpServers,omitempty"`
	PreferredNotif string                     `json:"preferredNotifyMethod,omitempty"`
}

// Settings aggregates all resolved configuration.
type Settings struct {
	User    UserConfig
	Project ProjectConfig
}
