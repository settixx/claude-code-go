package todo

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "TodoWrite"

var validStatuses = map[string]bool{
	"pending":     true,
	"in_progress": true,
	"completed":   true,
	"cancelled":   true,
}

type TodoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type Tool struct {
	toolutil.BaseTool
	mu    sync.Mutex
	items map[string]TodoItem
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"todo_write", "todo"},
			ToolSearchHint:  "manage todo list, task tracking, progress",
			ReadOnly:        false,
			ConcurrencySafe: false,
		},
		items: make(map[string]TodoItem),
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Create and manage a structured task list. " +
		"Supports pending, in_progress, completed, and cancelled statuses. " +
		"Use merge=true to update existing items or merge=false to replace all.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":      map[string]interface{}{"type": "string"},
						"content": map[string]interface{}{"type": "string"},
						"status":  map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "completed", "cancelled"}},
					},
					"required": []string{"id", "content", "status"},
				},
				"description": "Array of TODO items to create or update.",
			},
			"merge": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, merge into existing todos. If false, replace all.",
			},
		},
		Required: []string{"todos"},
	}
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	merge := toolutil.OptionalBool(input, "merge", false)

	rawTodos, ok := input["todos"]
	if !ok {
		return nil, fmt.Errorf("missing required field \"todos\"")
	}
	todoSlice, ok := rawTodos.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field \"todos\" must be an array")
	}

	items, err := parseTodos(todoSlice)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !merge {
		t.items = make(map[string]TodoItem, len(items))
	}

	for _, item := range items {
		if merge {
			existing, exists := t.items[item.ID]
			if exists {
				if item.Content != "" {
					existing.Content = item.Content
				}
				if item.Status != "" {
					existing.Status = item.Status
				}
				t.items[item.ID] = existing
				continue
			}
		}
		t.items[item.ID] = item
	}

	return &types.ToolResult{Data: t.formatSummary()}, nil
}

func parseTodos(raw []interface{}) ([]TodoItem, error) {
	items := make([]TodoItem, 0, len(raw))
	for i, entry := range raw {
		m, ok := entry.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("todos[%d] must be an object", i)
		}

		id, _ := m["id"].(string)
		content, _ := m["content"].(string)
		status, _ := m["status"].(string)

		if id == "" {
			return nil, fmt.Errorf("todos[%d]: missing \"id\"", i)
		}
		if status != "" && !validStatuses[status] {
			return nil, fmt.Errorf("todos[%d]: invalid status %q", i, status)
		}

		items = append(items, TodoItem{ID: id, Content: content, Status: status})
	}
	return items, nil
}

func (t *Tool) formatSummary() string {
	if len(t.items) == 0 {
		return "Todo list is empty."
	}

	var b strings.Builder
	b.WriteString("Current TODO list:\n")
	for _, item := range t.items {
		marker := statusMarker(item.Status)
		fmt.Fprintf(&b, "  %s [%s] %s (id: %s)\n", marker, item.Status, item.Content, item.ID)
	}
	return b.String()
}

func statusMarker(status string) string {
	switch status {
	case "completed":
		return "[x]"
	case "in_progress":
		return "[~]"
	case "cancelled":
		return "[-]"
	default:
		return "[ ]"
	}
}
