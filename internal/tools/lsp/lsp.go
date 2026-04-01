package lsp

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

var validActions = map[string]bool{
	"diagnostics": true,
	"hover":       true,
	"definition":  true,
	"references":  true,
}

// Tool is a stub for language-server-protocol integration.
type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "LSP",
			ToolAliases:     []string{"lsp", "language_server"},
			ToolSearchHint:  "lsp language server diagnostics hover definition references",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Query the Language Server Protocol for diagnostics, hover info, " +
		"go-to-definition, or find-references. (Stub — not yet connected.)", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"diagnostics", "hover", "definition", "references"},
				"description": "The LSP action to perform.",
			},
			"file": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file.",
			},
			"line": map[string]interface{}{
				"type":        "integer",
				"description": "1-based line number.",
			},
			"column": map[string]interface{}{
				"type":        "integer",
				"description": "1-based column number.",
			},
		},
		Required: []string{"action", "file"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	action, err := toolutil.RequireString(input, "action")
	if err != nil {
		return nil, err
	}
	if !validActions[action] {
		return nil, fmt.Errorf("invalid action %q; must be one of: diagnostics, hover, definition, references", action)
	}

	_, err = toolutil.RequireString(input, "file")
	if err != nil {
		return nil, err
	}

	return &types.ToolResult{
		Data: "LSP integration not yet configured. " +
			"This tool will be available once a language server is connected.",
	}, nil
}
