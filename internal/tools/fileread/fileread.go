package fileread

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName       = "FileRead"
	maxResultChars = 500_000
)

// Tool reads a file from the local filesystem.
type Tool struct {
	toolutil.BaseTool
}

// New creates a ready-to-use FileReadTool.
func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"Read", "file_read", "ReadFile"},
			ToolSearchHint:  "read files, view file contents",
			ToolMaxChars:    maxResultChars,
			ReadOnly:        true,
			Destructive:     false,
			ConcurrencySafe: true,
		},
	}
}

// Description returns a human-readable description for the model.
func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Read a file from the local filesystem. " +
		"Output lines are numbered as LINE_NUMBER|LINE_CONTENT. " +
		"Use offset and limit to read specific portions of large files.", nil
}

// InputSchema returns the JSON Schema for the tool's input.
func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "The 1-based line number to start reading from (default: 1)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "The number of lines to read (default: entire file)",
			},
		},
		Required: []string{"file_path"},
	}
}

// Call reads the file and returns its contents with line numbers.
func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	filePath, err := toolutil.RequireString(input, "file_path")
	if err != nil {
		return nil, err
	}

	offset := toolutil.OptionalInt(input, "offset", 1)
	if offset < 1 {
		offset = 1
	}
	limit := toolutil.OptionalInt(input, "limit", -1)

	lines, totalLines, err := readLines(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	if totalLines == 0 {
		return &types.ToolResult{Data: "File is empty."}, nil
	}

	startIdx := offset - 1
	if startIdx >= totalLines {
		msg := fmt.Sprintf(
			"Offset %d exceeds file length (%d lines). "+
				"Use a smaller offset or omit it to read from the beginning.",
			offset, totalLines,
		)
		return &types.ToolResult{Data: msg}, nil
	}

	endIdx := totalLines
	if limit > 0 {
		endIdx = startIdx + limit
		if endIdx > totalLines {
			endIdx = totalLines
		}
	}

	formatted := formatWithLineNumbers(lines[startIdx:endIdx], offset)
	result := toolutil.TruncateResult(formatted, maxResultChars)

	return &types.ToolResult{Data: result}, nil
}

func readLines(path string) ([]string, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	return lines, len(lines), nil
}

func formatWithLineNumbers(lines []string, startLine int) string {
	if len(lines) == 0 {
		return ""
	}

	maxLineNum := startLine + len(lines) - 1
	width := len(fmt.Sprintf("%d", maxLineNum))
	if width < 6 {
		width = 6
	}

	var sb strings.Builder
	for i, line := range lines {
		lineNum := startLine + i
		fmt.Fprintf(&sb, "%*d|%s\n", width, lineNum, line)
	}
	return sb.String()
}
