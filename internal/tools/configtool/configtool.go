package configtool

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const toolName = "Config"

type Tool struct {
	toolutil.BaseTool
	mu     sync.RWMutex
	values map[string]string
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"config"},
			ToolSearchHint:  "configuration, settings, get set list config values",
			ReadOnly:        false,
			ConcurrencySafe: false,
		},
		values: make(map[string]string),
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Read, write, or list configuration values. " +
		"Supports actions: get, set, list.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"get", "set", "list"},
				"description": "The config operation: get, set, or list.",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "Configuration key (required for get/set).",
			},
			"value": map[string]interface{}{
				"type":        "string",
				"description": "Value to set (required for set action).",
			},
		},
		Required: []string{"action"},
	}
}

func (t *Tool) IsReadOnly(input map[string]interface{}) bool {
	action := toolutil.OptionalString(input, "action", "")
	return action == "get" || action == "list"
}

func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	action, err := toolutil.RequireString(input, "action")
	if err != nil {
		return nil, err
	}

	switch action {
	case "get":
		return t.doGet(input)
	case "set":
		return t.doSet(input)
	case "list":
		return t.doList()
	default:
		return nil, fmt.Errorf("unknown action %q; must be get, set, or list", action)
	}
}

func (t *Tool) doGet(input map[string]interface{}) (*types.ToolResult, error) {
	key, err := toolutil.RequireString(input, "key")
	if err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	val, ok := t.values[key]
	if !ok {
		return &types.ToolResult{Data: fmt.Sprintf("Key %q is not set.", key)}, nil
	}
	return &types.ToolResult{Data: fmt.Sprintf("%s = %s", key, val)}, nil
}

func (t *Tool) doSet(input map[string]interface{}) (*types.ToolResult, error) {
	key, err := toolutil.RequireString(input, "key")
	if err != nil {
		return nil, err
	}
	value, err := toolutil.RequireString(input, "value")
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.values[key] = value
	return &types.ToolResult{Data: fmt.Sprintf("Set %s = %s", key, value)}, nil
}

func (t *Tool) doList() (*types.ToolResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.values) == 0 {
		return &types.ToolResult{Data: "No configuration values set."}, nil
	}

	keys := make([]string, 0, len(t.values))
	for k := range t.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s = %s\n", k, t.values[k])
	}
	return &types.ToolResult{Data: strings.TrimRight(b.String(), "\n")}, nil
}
