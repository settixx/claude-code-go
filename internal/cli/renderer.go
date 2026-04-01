package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// TUIRenderer implements interfaces.Renderer by forwarding messages as
// StreamEvents into a channel that the TUI consumes. The channel reference
// is swapped per query via SetEvents / ClearEvents.
type TUIRenderer struct {
	mu     sync.Mutex
	events chan<- types.StreamEvent
}

func NewTUIRenderer() *TUIRenderer {
	return &TUIRenderer{}
}

// SetEvents attaches the event sink for the current query.
func (r *TUIRenderer) SetEvents(ch chan<- types.StreamEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = ch
}

// ClearEvents detaches the current event sink.
func (r *TUIRenderer) ClearEvents() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = nil
}

func (r *TUIRenderer) send(ev types.StreamEvent) {
	r.mu.Lock()
	ch := r.events
	r.mu.Unlock()
	if ch != nil {
		ch <- ev
	}
}

func (r *TUIRenderer) RenderMessage(msg types.Message) {
	switch msg.Type {
	case types.MsgProgress:
		r.send(types.StreamEvent{
			Type:  types.EventContentBlockDelta,
			Delta: &types.DeltaBlock{Type: "text_delta", Text: msg.Text},
		})

	case types.MsgAssistant:
		if msg.APIMessage == nil {
			return
		}
		for i, block := range msg.APIMessage.Content {
			if block.Type == types.ContentText {
				r.send(types.StreamEvent{
					Type:         types.EventContentBlockStart,
					Index:        i,
					ContentBlock: &block,
				})
			}
			if block.Type == types.ContentToolUse {
				r.send(types.StreamEvent{
					Type:         types.EventContentBlockStart,
					Index:        i,
					ContentBlock: &block,
				})
			}
		}
	}
}

func (r *TUIRenderer) RenderError(err error) {
	r.send(types.StreamEvent{
		Type: types.EventError,
		Error: &struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{Type: "engine_error", Message: err.Error()},
	})
}

func (r *TUIRenderer) RenderSpinner(_ string) {}
func (r *TUIRenderer) StopSpinner()           {}

// ---------------------------------------------------------------------------
// StdRenderer — simple stderr/stdout renderer for non-interactive (print) mode
// ---------------------------------------------------------------------------

// StdRenderer implements interfaces.Renderer by writing directly to
// stdout / stderr. Used for print mode and simple output.
type StdRenderer struct{}

func NewStdRenderer() *StdRenderer { return &StdRenderer{} }

func (r *StdRenderer) RenderMessage(msg types.Message) {
	switch msg.Type {
	case types.MsgProgress:
		fmt.Fprint(os.Stdout, msg.Text)
	case types.MsgAssistant:
		if msg.APIMessage == nil {
			return
		}
		for _, block := range msg.APIMessage.Content {
			if block.Type == types.ContentText && block.Text != "" {
				fmt.Fprintln(os.Stdout, block.Text)
			}
		}
	case types.MsgSystem:
		fmt.Fprintf(os.Stderr, "[system] %s\n", msg.Text)
	}
}

func (r *StdRenderer) RenderError(err error) {
	fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
}

func (r *StdRenderer) RenderSpinner(text string) {
	fmt.Fprintf(os.Stderr, "⏳ %s\n", text)
}

func (r *StdRenderer) StopSpinner() {}

// ---------------------------------------------------------------------------
// JSONStreamRenderer — emits one JSON line per event (stream-json format)
// ---------------------------------------------------------------------------

type JSONStreamRenderer struct{}

func NewJSONStreamRenderer() *JSONStreamRenderer { return &JSONStreamRenderer{} }

func (r *JSONStreamRenderer) RenderMessage(msg types.Message) {
	switch msg.Type {
	case types.MsgProgress:
		line, _ := json.Marshal(map[string]interface{}{
			"type":  "content_block_delta",
			"delta": map[string]string{"text": msg.Text},
		})
		fmt.Fprintln(os.Stdout, string(line))

	case types.MsgAssistant:
		if msg.APIMessage == nil {
			return
		}
		line, _ := json.Marshal(map[string]interface{}{
			"type":    "message",
			"message": msg.APIMessage,
		})
		fmt.Fprintln(os.Stdout, string(line))
	}
}

func (r *JSONStreamRenderer) RenderError(err error) {
	line, _ := json.Marshal(map[string]interface{}{
		"type":  "error",
		"error": map[string]string{"message": err.Error()},
	})
	fmt.Fprintln(os.Stdout, string(line))
}

func (r *JSONStreamRenderer) RenderSpinner(_ string) {}
func (r *JSONStreamRenderer) StopSpinner()           {}

// ---------------------------------------------------------------------------
// JSONCollectRenderer — buffers all output, flushed as a single JSON object
// ---------------------------------------------------------------------------

type JSONCollectRenderer struct {
	mu       sync.Mutex
	text     strings.Builder
	messages []*types.APIMessage
	errors   []string
}

func NewJSONCollectRenderer() *JSONCollectRenderer { return &JSONCollectRenderer{} }

func (r *JSONCollectRenderer) RenderMessage(msg types.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch msg.Type {
	case types.MsgProgress:
		r.text.WriteString(msg.Text)
	case types.MsgAssistant:
		if msg.APIMessage != nil {
			r.messages = append(r.messages, msg.APIMessage)
		}
	}
}

func (r *JSONCollectRenderer) RenderError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, err.Error())
}

func (r *JSONCollectRenderer) RenderSpinner(_ string) {}
func (r *JSONCollectRenderer) StopSpinner()           {}

// Flush writes the collected output as a single JSON object to stdout.
func (r *JSONCollectRenderer) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := map[string]interface{}{
		"result": r.text.String(),
	}
	if len(r.messages) > 0 {
		out["messages"] = r.messages
	}
	if len(r.errors) > 0 {
		out["errors"] = r.errors
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}
