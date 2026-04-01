package agent

import (
	"context"
	"fmt"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// SendMessageTool routes messages to running agents by ID or name.
type SendMessageTool struct {
	toolutil.BaseTool
	pool *coordinator.WorkerPool
}

// NewSendMessageTool creates a SendMessageTool wired to the given pool.
func NewSendMessageTool(pool *coordinator.WorkerPool) *SendMessageTool {
	return &SendMessageTool{
		BaseTool: toolutil.BaseTool{
			ToolName:        "SendMessage",
			ToolAliases:     []string{"SendMessageToAgent"},
			ToolSearchHint:  "send message agent communicate",
			ConcurrencySafe: true,
		},
		pool: pool,
	}
}

// Description returns the tool description for the LLM.
func (t *SendMessageTool) Description(_ map[string]interface{}) (string, error) {
	return "Send a message to a running agent by ID or name. " +
		"Use this to communicate with background agents or provide follow-up instructions.", nil
}

// InputSchema returns the JSON Schema for SendMessage input.
func (t *SendMessageTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"to": map[string]interface{}{
				"type":        "string",
				"description": "Agent ID or human-readable name to send the message to.",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "The message content to deliver to the agent.",
			},
		},
		Required: []string{"to", "message"},
	}
}

// IsReadOnly returns false — sending messages can trigger side effects.
func (t *SendMessageTool) IsReadOnly(_ map[string]interface{}) bool { return false }

// Call sends a message to the specified agent.
func (t *SendMessageTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	to, err := toolutil.RequireString(input, "to")
	if err != nil {
		return nil, err
	}
	message, err := toolutil.RequireString(input, "message")
	if err != nil {
		return nil, err
	}

	worker := t.resolveWorker(to)
	if worker == nil {
		return nil, fmt.Errorf("no agent found with ID or name %q", to)
	}

	msg := types.Message{
		Type: types.MsgUser,
		Role: "user",
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: message,
		}},
		Origin: &types.MessageOrigin{Kind: types.OriginCoordinator},
	}

	worker.Send(msg)

	return &types.ToolResult{
		Data: fmt.Sprintf("Message delivered to agent %q (%s).", worker.Name, worker.ID),
	}, nil
}

func (t *SendMessageTool) resolveWorker(to string) *coordinator.Worker {
	if id, ok := types.ToAgentId(to); ok {
		w, found := t.pool.Get(id)
		if found {
			return w
		}
	}
	w, found := t.pool.GetByName(to)
	if found {
		return w
	}
	return nil
}
