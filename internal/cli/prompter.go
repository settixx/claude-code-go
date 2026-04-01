package cli

import (
	"context"

	"github.com/settixx/claude-code-go/internal/permissions"
	"github.com/settixx/claude-code-go/internal/tui"
)

// TUIPrompter adapts the TUI App's permission dialog into a PermissionPrompter.
type TUIPrompter struct {
	app *tui.App
}

// NewTUIPrompter creates a TUI-based permission prompter.
func NewTUIPrompter(app *tui.App) *TUIPrompter {
	return &TUIPrompter{app: app}
}

// Prompt presents a permission dialog in the TUI and blocks until the user responds.
func (p *TUIPrompter) Prompt(ctx context.Context, req permissions.PermissionRequest) (permissions.PermissionChoice, error) {
	select {
	case <-ctx.Done():
		return permissions.ChoiceDeny, ctx.Err()
	default:
	}

	description := req.Description
	if description == "" {
		description = "Run tool " + req.ToolName
	}

	if p.app.SendPermissionRequest(req.ToolName, description) {
		return permissions.ChoiceAllow, nil
	}
	return permissions.ChoiceDeny, nil
}
