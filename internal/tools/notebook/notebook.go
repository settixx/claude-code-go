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
	return "Edit or insert a cell in a Jupyter notebook (.ipynb file). " +
		"Set is_new_cell to true to insert a new cell at the given index.", nil
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
				"description": "0-based index of the cell to edit or insert at.",
			},
			"new_source": map[string]interface{}{
				"type":        "string",
				"description": "The new source content for the cell.",
			},
			"is_new_cell": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, insert a new cell at cell_index instead of editing an existing one.",
			},
			"cell_language": map[string]interface{}{
				"type":        "string",
				"description": "Language/type for new cells: 'python', 'markdown', 'raw', etc. Defaults to 'python'.",
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
	isNewCell := toolutil.OptionalBool(input, "is_new_cell", false)
	cellLang := toolutil.OptionalString(input, "cell_language", "python")

	nb, err := readNotebook(nbPath)
	if err != nil {
		return &types.ToolResult{Data: fmt.Sprintf("Failed to read notebook: %v", err)}, nil
	}

	if isNewCell {
		return insertNewCell(nb, nbPath, cellIndex, newSource, cellLang)
	}
	return editExistingCell(nb, nbPath, cellIndex, newSource)
}

func editExistingCell(nb *notebook, nbPath string, cellIndex int, newSource string) (*types.ToolResult, error) {
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

func insertNewCell(nb *notebook, nbPath string, cellIndex int, newSource, cellLang string) (*types.ToolResult, error) {
	if cellIndex > len(nb.Cells) {
		cellIndex = len(nb.Cells)
	}

	cellType := resolveCellType(cellLang)
	newCell := cell{
		CellType: cellType,
		Source:   splitSourceLines(newSource),
		Metadata: map[string]interface{}{},
	}
	if cellType == "code" {
		newCell.Outputs = []interface{}{}
	}

	cells := make([]cell, 0, len(nb.Cells)+1)
	cells = append(cells, nb.Cells[:cellIndex]...)
	cells = append(cells, newCell)
	cells = append(cells, nb.Cells[cellIndex:]...)
	nb.Cells = cells

	if err := writeNotebook(nbPath, nb); err != nil {
		return &types.ToolResult{Data: fmt.Sprintf("Failed to write notebook: %v", err)}, nil
	}

	return &types.ToolResult{
		Data: fmt.Sprintf("Inserted new %s cell at index %d in %s", cellType, cellIndex, nbPath),
	}, nil
}

func resolveCellType(lang string) string {
	switch strings.ToLower(lang) {
	case "markdown":
		return "markdown"
	case "raw":
		return "raw"
	default:
		return "code"
	}
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
