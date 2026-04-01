package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "NotebookEdit"

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"notebook_edit"},
			ToolSearchHint:  "edit jupyter notebook cell, ipynb",
			ReadOnly:        false,
			ConcurrencySafe: false,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Edit a cell in a Jupyter notebook (.ipynb file). " +
		"Reads the notebook JSON, modifies the specified cell's source, and writes it back.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"notebook_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the .ipynb notebook file.",
			},
			"cell_index": map[string]interface{}{
				"type":        "integer",
				"description": "0-based index of the cell to edit.",
			},
			"new_source": map[string]interface{}{
				"type":        "string",
				"description": "The new source content for the cell.",
			},
		},
		Required: []string{"notebook_path", "cell_index", "new_source"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	nbPath, err := toolutil.RequireString(input, "notebook_path")
	if err != nil {
		return nil, err
	}
	cellIndex := toolutil.OptionalInt(input, "cell_index", -1)
	if cellIndex < 0 {
		return nil, fmt.Errorf("missing or invalid required field \"cell_index\"")
	}
	newSource, err := toolutil.RequireString(input, "new_source")
	if err != nil {
		return nil, err
	}

	nb, err := readNotebook(nbPath)
	if err != nil {
		return &types.ToolResult{Data: fmt.Sprintf("Failed to read notebook: %v", err)}, nil
	}

	if cellIndex >= len(nb.Cells) {
		return &types.ToolResult{
			Data: fmt.Sprintf("Cell index %d out of range (notebook has %d cells)", cellIndex, len(nb.Cells)),
		}, nil
	}

	nb.Cells[cellIndex].Source = splitSourceLines(newSource)

	if err := writeNotebook(nbPath, nb); err != nil {
		return &types.ToolResult{Data: fmt.Sprintf("Failed to write notebook: %v", err)}, nil
	}

	return &types.ToolResult{
		Data: fmt.Sprintf("Updated cell %d in %s", cellIndex, nbPath),
	}, nil
}

type notebook struct {
	Cells         []cell                 `json:"cells"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	NBFormat      int                    `json:"nbformat"`
	NBFormatMinor int                    `json:"nbformat_minor"`
}

type cell struct {
	CellType       string                 `json:"cell_type"`
	Source         []string               `json:"source"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Outputs       []interface{}          `json:"outputs,omitempty"`
	ExecutionCount *int                   `json:"execution_count,omitempty"`
}

func readNotebook(path string) (*notebook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var nb notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, fmt.Errorf("invalid notebook JSON: %w", err)
	}
	return &nb, nil
}

func writeNotebook(path string, nb *notebook) error {
	data, err := json.MarshalIndent(nb, "", " ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func splitSourceLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		if i < len(lines)-1 {
			result[i] = line + "\n"
		} else {
			result[i] = line
		}
	}
	return result
}
