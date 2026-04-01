package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// TermRenderer implements interfaces.Renderer using ANSI escape codes.
type TermRenderer struct {
	spinner *Spinner
}

// NewTermRenderer creates a ready-to-use terminal renderer.
func NewTermRenderer() *TermRenderer {
	return &TermRenderer{}
}

// RenderMessage formats and prints a conversation message to stdout.
func (r *TermRenderer) RenderMessage(msg types.Message) {
	switch msg.Type {
	case types.MsgUser:
		r.renderUserMessage(msg)
	case types.MsgAssistant:
		r.renderAssistantMessage(msg)
	case types.MsgSystem:
		r.renderSystemMessage(msg)
	case types.MsgProgress:
		r.renderProgressMessage(msg)
	default:
		fmt.Fprintln(os.Stdout, Dim(fmt.Sprintf("[%s message]", msg.Type)))
	}
}

// RenderError prints an error in red to stdout.
func (r *TermRenderer) RenderError(err error) {
	r.ensureSpinnerStopped()
	fmt.Fprintln(os.Stdout, Red("✗ Error: "+err.Error()))
}

// RenderSpinner starts or updates the spinner with new text.
func (r *TermRenderer) RenderSpinner(text string) {
	if r.spinner != nil && r.spinner.running {
		r.spinner.UpdateText(text)
		return
	}
	r.spinner = NewSpinner(text)
	r.spinner.Start()
}

// StopSpinner halts any running spinner and clears its line.
func (r *TermRenderer) StopSpinner() {
	r.ensureSpinnerStopped()
}

func (r *TermRenderer) ensureSpinnerStopped() {
	if r.spinner == nil {
		return
	}
	r.spinner.Stop()
	r.spinner = nil
}

func (r *TermRenderer) renderUserMessage(msg types.Message) {
	text := extractText(msg)
	if text == "" {
		return
	}
	fmt.Fprintln(os.Stdout, Cyan("> "+text))
}

func (r *TermRenderer) renderAssistantMessage(msg types.Message) {
	r.ensureSpinnerStopped()

	if msg.IsAPIError {
		fmt.Fprintln(os.Stdout, Red("API Error: "+msg.APIError))
		return
	}

	blocks := extractBlocks(msg)
	for i, block := range blocks {
		if i > 0 {
			fmt.Fprintln(os.Stdout)
		}
		fmt.Fprintln(os.Stdout, FormatContentBlock(block))
	}
}

func (r *TermRenderer) renderSystemMessage(msg types.Message) {
	text := extractText(msg)
	if text == "" {
		return
	}

	switch msg.Level {
	case types.LevelWarning:
		fmt.Fprintln(os.Stdout, Yellow("⚠ "+text))
	case types.LevelError:
		fmt.Fprintln(os.Stdout, Red("✗ "+text))
	default:
		fmt.Fprintln(os.Stdout, Blue("ℹ "+text))
	}
}

func (r *TermRenderer) renderProgressMessage(msg types.Message) {
	text := extractText(msg)
	if text != "" {
		r.RenderSpinner(text)
	}
}

// extractText returns the best text representation for a message.
func extractText(msg types.Message) string {
	if msg.Text != "" {
		return msg.Text
	}
	var parts []string
	for _, b := range msg.Content {
		if b.Type == types.ContentText && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}
	if msg.APIMessage != nil {
		for _, b := range msg.APIMessage.Content {
			if b.Type == types.ContentText && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// extractBlocks returns the content blocks from either inline or the embedded APIMessage.
func extractBlocks(msg types.Message) []types.ContentBlock {
	if len(msg.Content) > 0 {
		return msg.Content
	}
	if msg.APIMessage != nil {
		return msg.APIMessage.Content
	}
	return nil
}
