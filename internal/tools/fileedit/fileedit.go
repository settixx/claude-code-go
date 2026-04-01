package fileedit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName       = "FileEdit"
	maxResultChars = 100_000
)

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"Edit", "file_edit", "StrReplace"},
			ToolSearchHint:  "modify file contents in place",
			ToolMaxChars:    maxResultChars,
			ReadOnly:        false,
			Destructive:     false,
			ConcurrencySafe: false,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Edit a file by replacing an exact string match (or regex pattern) with new content. " +
		"The old_string must match exactly (including whitespace and indentation) unless regex is true. " +
		"Set replace_all to true to replace all occurrences.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to edit",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "The exact string to find in the file (or a regex pattern if regex is true)",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The replacement string",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences (default false)",
			},
			"regex": map[string]interface{}{
				"type":        "boolean",
				"description": "Treat old_string as a regular expression pattern",
			},
		},
		Required: []string{"file_path", "old_string", "new_string"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	filePath, err := toolutil.RequireString(input, "file_path")
	if err != nil {
		return nil, err
	}
	oldString, err := toolutil.RequireString(input, "old_string")
	if err != nil {
		return nil, err
	}
	newString, err := toolutil.RequireString(input, "new_string")
	if err != nil {
		return nil, err
	}
	replaceAll := toolutil.OptionalBool(input, "replace_all", false)
	useRegex := toolutil.OptionalBool(input, "regex", false)

	if !useRegex && oldString == newString {
		return nil, fmt.Errorf("old_string and new_string are identical; no changes to make")
	}

	content, isNew, err := readOrCreateFile(filePath, oldString)
	if err != nil {
		return nil, err
	}

	if isNew {
		return writeNewFile(filePath, newString)
	}

	if useRegex {
		return regexReplace(filePath, content, oldString, newString, replaceAll)
	}
	return literalReplace(filePath, content, oldString, newString, replaceAll)
}

func literalReplace(filePath, content, oldString, newString string, replaceAll bool) (*types.ToolResult, error) {
	count := strings.Count(content, oldString)
	if count == 0 {
		return nil, fmt.Errorf("old_string not found in %s", filePath)
	}
	if count > 1 && !replaceAll {
		return nil, fmt.Errorf(
			"found %d occurrences of old_string in %s but replace_all is false; "+
				"provide more context to uniquely identify the target or set replace_all to true",
			count, filePath,
		)
	}

	var updated string
	if replaceAll {
		updated = strings.ReplaceAll(content, oldString, newString)
	} else {
		updated = strings.Replace(content, oldString, newString, 1)
	}

	if err := os.WriteFile(filePath, []byte(updated), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file %q: %w", filePath, err)
	}

	suffix := ""
	if replaceAll && count > 1 {
		suffix = fmt.Sprintf(" All %d occurrences replaced.", count)
	}
	msg := fmt.Sprintf("The file %s has been updated successfully.%s", filePath, suffix)
	return &types.ToolResult{Data: msg}, nil
}

func regexReplace(filePath, content, pattern, replacement string, replaceAll bool) (*types.ToolResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	matches := re.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("regex pattern %q not found in %s", pattern, filePath)
	}

	var updated string
	if replaceAll {
		updated = re.ReplaceAllString(content, replacement)
	} else {
		replaced := false
		updated = re.ReplaceAllStringFunc(content, func(match string) string {
			if replaced {
				return match
			}
			replaced = true
			return re.ReplaceAllString(match, replacement)
		})
	}

	if err := os.WriteFile(filePath, []byte(updated), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file %q: %w", filePath, err)
	}

	count := len(matches)
	suffix := ""
	if replaceAll && count > 1 {
		suffix = fmt.Sprintf(" All %d matches replaced.", count)
	} else {
		suffix = fmt.Sprintf(" %d match(es) found, 1 replaced.", count)
	}
	msg := fmt.Sprintf("The file %s has been updated successfully (regex mode).%s", filePath, suffix)
	return &types.ToolResult{Data: msg}, nil
}

func readOrCreateFile(filePath, oldString string) (string, bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) && oldString == "" {
			return "", true, nil
		}
		return "", false, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}
	return string(data), false, nil
}

func writeNewFile(filePath, content string) (*types.ToolResult, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", dir, err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file %q: %w", filePath, err)
	}
	msg := fmt.Sprintf("File created successfully at: %s", filePath)
	return &types.ToolResult{Data: msg}, nil
}
