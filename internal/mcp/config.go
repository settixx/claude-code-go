package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/settixx/claude-code-go/internal/types"
)

// DefaultConfigPath returns ~/.claude/mcp_servers.json.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "mcp_servers.json"), nil
}

// mcpConfigFile mirrors the JSON structure of the config file.
type mcpConfigFile struct {
	MCPServers map[string]types.McpServerConfig `json:"mcpServers"`
}

// LoadMCPConfig reads MCP server configurations from the given path.
// Returns an empty map (not an error) when the file does not exist.
func LoadMCPConfig(cfgPath string) (map[string]types.McpServerConfig, error) {
	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return make(map[string]types.McpServerConfig), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}

	var cfg mcpConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", cfgPath, err)
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]types.McpServerConfig)
	}
	return cfg.MCPServers, nil
}

// SaveMCPConfig writes MCP server configurations to the given path, creating
// parent directories as needed.
func SaveMCPConfig(cfgPath string, servers map[string]types.McpServerConfig) error {
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfg := mcpConfigFile{MCPServers: servers}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", cfgPath, err)
	}
	return nil
}
