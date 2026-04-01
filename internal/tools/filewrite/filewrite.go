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

type Tool struct {
	toolutil.BaseTool
}

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

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Write content to a file on the local filesystem. " +
		"Creates the file and any parent directories if they don't exist. " +
		"Use mode 'append' to add to the end instead of overwriting.", nil
}

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
			"mode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"write", "append"},
				"description": "Write mode: 'write' (default, overwrite) or 'append' (add to end)",
			},
		},
		Required: []string{"file_path", "content"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	filePath, err := toolutil.RequireString(input, "file_path")
	if err != nil {
		return nil, err
	}
	content, err := toolutil.RequireString(input, "content")
	if err != nil {
		return nil, err
	}
	mode := toolutil.OptionalString(input, "mode", "write")

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	if mode == "append" {
		return appendToFile(filePath, content)
	}
	return overwriteFile(filePath, content)
}

func overwriteFile(filePath, content string) (*types.ToolResult, error) {
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

func appendToFile(filePath, content string) (*types.ToolResult, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q for append: %w", filePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return nil, fmt.Errorf("failed to append to file %q: %w", filePath, err)
	}

	msg := fmt.Sprintf("Content appended successfully to: %s", filePath)
	return &types.ToolResult{Data: msg}, nil
}
