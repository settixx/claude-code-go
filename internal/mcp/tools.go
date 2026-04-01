package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

const maxToolResultChars = 100_000

// MCPToolAdapter wraps an MCPToolSchema + Client so it satisfies types.Tool.
// The tool name is prefixed as mcp__<serverName>__<toolName>.
type MCPToolAdapter struct {
	schema     MCPToolSchema
	serverName string
	client     *Client
}

// NewMCPToolAdapter creates a Tool adapter for a remote MCP tool.
func NewMCPToolAdapter(schema MCPToolSchema, serverName string, client *Client) *MCPToolAdapter {
	return &MCPToolAdapter{schema: schema, serverName: serverName, client: client}
}

// Name returns the namespaced tool identifier: mcp__<server>__<tool>.
func (a *MCPToolAdapter) Name() string {
	return fmt.Sprintf("mcp__%s__%s", a.serverName, a.schema.Name)
}

// Aliases returns no aliases.
func (a *MCPToolAdapter) Aliases() []string { return nil }

// Description returns the tool description from the server.
func (a *MCPToolAdapter) Description(_ map[string]interface{}) (string, error) {
	return a.schema.Description, nil
}

// InputSchema converts the MCP input schema to a types.ToolInputSchema.
func (a *MCPToolAdapter) InputSchema() types.ToolInputSchema {
	schema := types.ToolInputSchema{Type: "object"}
	if a.schema.InputSchema == nil {
		return schema
	}

	raw, err := json.Marshal(a.schema.InputSchema)
	if err != nil {
		return schema
	}
	var parsed struct {
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties,omitempty"`
		Required   []string               `json:"required,omitempty"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return schema
	}
	if parsed.Type != "" {
		schema.Type = parsed.Type
	}
	schema.Properties = parsed.Properties
	schema.Required = parsed.Required
	return schema
}

// Call delegates execution to the remote MCP server.
func (a *MCPToolAdapter) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	result, err := a.client.CallTool(ctx, a.schema.Name, input)
	if err != nil {
		return nil, fmt.Errorf("mcp call %s: %w", a.Name(), err)
	}
	text := truncateResult(extractText(result), a.Name())
	if result.IsError {
		return &types.ToolResult{Data: text}, fmt.Errorf("tool returned error: %s", text)
	}
	return &types.ToolResult{Data: text}, nil
}

func extractText(result *ToolCallResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	var parts []string
	for _, c := range result.Content {
		if c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

const truncationSuffix = "\n[truncated — result exceeded 100K chars]"

func truncateResult(text, toolName string) string {
	if len(text) <= maxToolResultChars {
		return text
	}
	log.Printf("mcp: tool %s result truncated from %d to %d chars", toolName, len(text), maxToolResultChars)
	return text[:maxToolResultChars-len(truncationSuffix)] + truncationSuffix
}

// IsEnabled always returns true for remote MCP tools.
func (a *MCPToolAdapter) IsEnabled() bool { return true }

// IsReadOnly returns false; we cannot infer read-only from the MCP schema.
func (a *MCPToolAdapter) IsReadOnly(_ map[string]interface{}) bool { return false }

// IsDestructive returns false by default.
func (a *MCPToolAdapter) IsDestructive(_ map[string]interface{}) bool { return false }

// IsConcurrencySafe returns false by default.
func (a *MCPToolAdapter) IsConcurrencySafe(_ map[string]interface{}) bool { return false }

// CheckPermissions asks the user for permission by default.
func (a *MCPToolAdapter) CheckPermissions(_ map[string]interface{}, _ types.PermissionMode) types.PermissionResult {
	return types.PermissionResult{Allowed: false, Reason: "MCP tool requires user approval"}
}

// MaxResultSizeChars returns a generous limit for MCP tool results.
func (a *MCPToolAdapter) MaxResultSizeChars() int { return 512 * 1024 }

// InterruptBehavior returns "cancel" so in-flight calls can be interrupted.
func (a *MCPToolAdapter) InterruptBehavior() string { return "cancel" }

// SearchHint returns the raw tool name for tool discovery.
func (a *MCPToolAdapter) SearchHint() string { return a.schema.Name }
