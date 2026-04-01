package filewrite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName       = "FileWrite"
	maxResultChars = 100_000
)

// Tool creates or overwrites files on the local filesystem.
type Tool struct {
	toolutil.BaseTool
}

// New creates a ready-to-use FileWriteTool.
func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"Write", "file_write", "WriteFile"},
			ToolSearchHint:  "create or overwrite files",
			ToolMaxChars:    maxResultChars,
			ReadOnly:        false,
			Destructive:     true,
			ConcurrencySafe: false,
		},
	}
}

// Description returns a human-readable description for the model.
func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Write content to a file on the local filesystem. " +
		"Creates the file and any parent directories if they don't exist. " +
		"Overwrites the file if it already exists.", nil
}

// InputSchema returns the JSON Schema for the tool's input.
func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		Required: []string{"file_path", "content"},
	}
}

// Call writes content to the file, creating parent directories as needed.
func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	filePath, err := toolutil.RequireString(input, "file_path")
	if err != nil {
		return nil, err
	}
	content, err := toolutil.RequireString(input, "content")
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	_, statErr := os.Stat(filePath)
	isNew := os.IsNotExist(statErr)

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file %q: %w", filePath, err)
	}

	action := "updated"
	if isNew {
		action = "created"
	}
	msg := fmt.Sprintf("File %s successfully at: %s", action, filePath)
	return &types.ToolResult{Data: msg}, nil
}
