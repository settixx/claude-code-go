package worktree

import (
	"context"
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/coordinator"
	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

// EnterTool creates a git worktree and switches the working context.
type EnterTool struct {
	toolutil.BaseTool
	baseCWD string
}

func NewEnterTool(baseCWD string) *EnterTool {
	return &EnterTool{
		BaseTool: toolutil.BaseTool{
			ToolName:       "EnterWorktree",
			ToolAliases:    []string{"enter_worktree"},
			ToolSearchHint: "worktree git branch isolate enter",
		},
		baseCWD: baseCWD,
	}
}

func (t *EnterTool) Description(_ map[string]interface{}) (string, error) {
	return "Create and enter a git worktree for isolated work on a branch.", nil
}

func (t *EnterTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"branch": map[string]interface{}{
				"type":        "string",
				"description": "Branch name for the worktree. Auto-generated if omitted.",
			},
		},
	}
}

func (t *EnterTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	branch := toolutil.OptionalString(input, "branch", "worktree-session")

	path, err := coordinator.CreateWorktree(t.baseCWD, branch)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	return &types.ToolResult{
		Data: fmt.Sprintf("Worktree created.\n  Branch: %s\n  Path: %s", branch, path),
	}, nil
}

// ExitTool leaves the current worktree, optionally cleaning it up.
type ExitTool struct {
	toolutil.BaseTool
}

func NewExitTool() *ExitTool {
	return &ExitTool{
		BaseTool: toolutil.BaseTool{
			ToolName:       "ExitWorktree",
			ToolAliases:    []string{"exit_worktree"},
			ToolSearchHint: "worktree git exit leave cleanup",
		},
	}
}

func (t *ExitTool) Description(_ map[string]interface{}) (string, error) {
	return "Exit the current git worktree. Optionally remove the worktree directory.", nil
}

func (t *ExitTool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the worktree to exit/remove.",
			},
			"cleanup": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, remove the worktree directory after exiting.",
			},
		},
		Required: []string{"path"},
	}
}

func (t *ExitTool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	path, err := toolutil.RequireString(input, "path")
	if err != nil {
		return nil, err
	}

	cleanup := toolutil.OptionalBool(input, "cleanup", false)
	if !cleanup {
		return &types.ToolResult{
			Data: fmt.Sprintf("Exited worktree at %s. Directory left in place.", path),
		}, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &types.ToolResult{
			Data: fmt.Sprintf("Worktree path %s does not exist; nothing to clean up.", path),
		}, nil
	}

	if err := coordinator.RemoveWorktree(path); err != nil {
		return nil, fmt.Errorf("remove worktree: %w", err)
	}
	return &types.ToolResult{
		Data: fmt.Sprintf("Worktree at %s removed.", path),
	}, nil
}
