package sleep

import (
	"context"
	"fmt"
	"time"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName   = "Sleep"
	maxSeconds = 300
)

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"sleep", "wait"},
			ToolSearchHint:  "sleep, wait, pause, delay execution",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Pause execution for the specified number of seconds. " +
		"Supports context cancellation. Maximum 300 seconds.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"seconds": map[string]interface{}{
				"type":        "integer",
				"description": "Number of seconds to sleep (1-300).",
			},
		},
		Required: []string{"seconds"},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	secs := toolutil.OptionalInt(input, "seconds", 0)
	if secs <= 0 {
		return nil, fmt.Errorf("seconds must be a positive integer")
	}
	if secs > maxSeconds {
		secs = maxSeconds
	}

	timer := time.NewTimer(time.Duration(secs) * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		return &types.ToolResult{Data: fmt.Sprintf("Slept for %d seconds.", secs)}, nil
	case <-ctx.Done():
		return &types.ToolResult{Data: fmt.Sprintf("Sleep interrupted after partial wait (requested %d seconds).", secs)}, nil
	}
}
