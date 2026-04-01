package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// Manager manages multiple MCP server connections and aggregates their tools
// and resources.
type Manager struct {
	mu           sync.RWMutex
	clients      map[string]*Client
	configs      map[string]types.McpServerConfig
	instructions map[string]string
}

// ServerInstruction pairs a server name with its instruction text.
type ServerInstruction struct {
	ServerName   string
	Instructions string
}

// NewManager creates an empty MCP server manager.
func NewManager() *Manager {
	return &Manager{
		clients:      make(map[string]*Client),
		configs:      make(map[string]types.McpServerConfig),
		instructions: make(map[string]string),
	}
}

// AddServer registers a server configuration. If a client with the same name
// already exists, it is disconnected first.
func (m *Manager) AddServer(name string, cfg types.McpServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, exists := m.clients[name]; exists {
		_ = old.Disconnect()
	}
	m.configs[name] = cfg
	m.clients[name] = NewClient(name, cfg)
	return nil
}

// RemoveServer disconnects and removes a server by name.
func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("server %q not found", name)
	}
	err := client.Disconnect()
	delete(m.clients, name)
	delete(m.configs, name)
	return err
}

// ConnectAll connects to every registered server. Errors are collected but
// do not prevent other servers from connecting.
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	snapshot := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		snapshot = append(snapshot, c)
	}
	m.mu.RUnlock()

	var errs []error
	for _, c := range snapshot {
		if err := c.Connect(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(errs)
}

// DisconnectAll shuts down every server connection.
func (m *Manager) DisconnectAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, c := range m.clients {
		if err := c.Disconnect(); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(errs)
}

// AllTools collects tools from all connected servers, wrapped as types.Tool.
func (m *Manager) AllTools() []types.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []types.Tool
	ctx := context.Background()
	for _, c := range m.clients {
		schemas, err := c.ListTools(ctx)
		if err != nil {
			continue
		}
		for _, s := range schemas {
			tools = append(tools, NewMCPToolAdapter(s, c.Name(), c))
		}
	}
	return tools
}

// AllResources collects resources from all connected servers, grouped by
// server name.
func (m *Manager) AllResources() map[string][]types.ServerResource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]types.ServerResource)
	ctx := context.Background()
	for _, c := range m.clients {
		resources, err := c.ListResources(ctx)
		if err != nil {
			continue
		}
		for _, r := range resources {
			result[c.Name()] = append(result[c.Name()], types.ServerResource{
				URI:         r.URI,
				Name:        r.Name,
				Description: r.Description,
				MimeType:    r.MimeType,
			})
		}
	}
	return result
}

// Connections returns the current connection status of every server.
func (m *Manager) Connections() []types.MCPServerConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]types.MCPServerConnection, 0, len(m.clients))
	for name, c := range m.clients {
		status := "disconnected"
		c.mu.Lock()
		if c.connected {
			status = "connected"
		}
		c.mu.Unlock()
		conns = append(conns, types.MCPServerConnection{Name: name, Status: status})
	}
	return conns
}

// RefreshTools re-fetches tools from all connected servers, updating internal
// state. Use this after a server signals that its tool list has changed.
func (m *Manager) RefreshTools(ctx context.Context) error {
	m.mu.RLock()
	snapshot := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		snapshot = append(snapshot, c)
	}
	m.mu.RUnlock()

	var errs []error
	for _, c := range snapshot {
		if _, err := c.ListTools(ctx); err != nil {
			errs = append(errs, fmt.Errorf("refresh tools for %q: %w", c.Name(), err))
		}
	}
	return joinErrors(errs)
}

// SetServerInstructions stores instruction text for the named server.
func (m *Manager) SetServerInstructions(serverName, instructions string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instructions[serverName] = instructions
}

// GetServerInstructions returns the instruction text for the named server.
func (m *Manager) GetServerInstructions(serverName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.instructions[serverName]
}

// AllInstructions returns instruction pairs for every server that has them.
func (m *Manager) AllInstructions() []ServerInstruction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]ServerInstruction, 0, len(m.instructions))
	for name, text := range m.instructions {
		if text == "" {
			continue
		}
		out = append(out, ServerInstruction{ServerName: name, Instructions: text})
	}
	return out
}

// FormatInstructionsText builds a markdown-formatted string from all
// server instructions, suitable for appending to a system prompt.
func (m *Manager) FormatInstructionsText() string {
	instructions := m.AllInstructions()
	if len(instructions) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("# MCP Server Instructions\n\n")
	for _, inst := range instructions {
		fmt.Fprintf(&b, "## %s\n%s\n\n", inst.ServerName, inst.Instructions)
	}
	return b.String()
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	msg := ""
	for i, e := range errs {
		if i > 0 {
			msg += "; "
		}
		msg += e.Error()
	}
	return fmt.Errorf("%s", msg)
}
