package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// streamReader parses a text/event-stream body into typed StreamEvents.
type streamReader struct {
	reader *bufio.Reader
}

func newStreamReader(r io.Reader) *streamReader {
	return &streamReader{reader: bufio.NewReaderSize(r, 64*1024)}
}

// sseRawEvent is a single raw SSE frame before JSON parsing.
type sseRawEvent struct {
	Event string
	Data  string
}

// readEvent reads the next complete SSE event from the stream.
// Returns io.EOF when the stream ends.
func (sr *streamReader) readEvent() (*sseRawEvent, error) {
	var event, data strings.Builder
	hasContent := false

	for {
		line, err := sr.reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		if line == "" && hasContent {
			return &sseRawEvent{
				Event: strings.TrimSpace(event.String()),
				Data:  strings.TrimSpace(data.String()),
			}, nil
		}

		if line == "" && err != nil {
			if hasContent {
				return &sseRawEvent{
					Event: strings.TrimSpace(event.String()),
					Data:  strings.TrimSpace(data.String()),
				}, nil
			}
			return nil, err
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "event:") {
			event.Reset()
			event.WriteString(strings.TrimPrefix(line, "event:"))
			hasContent = true
		} else if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteString("\n")
			}
			data.WriteString(strings.TrimPrefix(line, "data:"))
			hasContent = true
		}

		if err != nil {
			if hasContent {
				return &sseRawEvent{
					Event: strings.TrimSpace(event.String()),
					Data:  strings.TrimSpace(data.String()),
				}, nil
			}
			return nil, err
		}
	}
}

// parseStreamEvent converts a raw SSE event into a typed StreamEvent.
func parseStreamEvent(raw *sseRawEvent) (*types.StreamEvent, error) {
	if raw.Data == "" {
		return nil, fmt.Errorf("empty data field for event %q", raw.Event)
	}

	var evt types.StreamEvent
	if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
		return nil, fmt.Errorf("unmarshal event %q: %w", raw.Event, err)
	}

	if evt.Type == "" {
		evt.Type = types.StreamEventType(raw.Event)
	}
	return &evt, nil
}

// ReadSSEStream reads an SSE stream and pushes parsed events into a channel.
// The channel is closed when the stream ends or an error occurs.
func ReadSSEStream(body io.ReadCloser) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 16)
	go readSSELoop(body, ch)
	return ch
}

func readSSELoop(body io.ReadCloser, ch chan<- types.StreamEvent) {
	defer close(ch)
	defer body.Close()

	sr := newStreamReader(body)
	for {
		raw, err := sr.readEvent()
		if err != nil {
			if err != io.EOF {
				slog.Warn("SSE stream read error", "error", err)
				ch <- types.StreamEvent{
					Type: types.EventError,
					Error: &struct {
						Type    string `json:"type"`
						Message string `json:"message"`
					}{
						Type:    "stream_error",
						Message: err.Error(),
					},
				}
			}
			return
		}

		if raw.Event == "ping" || (raw.Event == "" && raw.Data == "") {
			continue
		}

		evt, parseErr := parseStreamEvent(raw)
		if parseErr != nil {
			slog.Warn("SSE event parse error", "error", parseErr, "raw_event", raw.Event)
			continue
		}

		ch <- *evt

		if evt.Type == types.EventMessageStop {
			return
		}
	}
}

// DeltaAssembler accumulates streaming deltas into a complete APIMessage.
type DeltaAssembler struct {
	message       *types.APIMessage
	contentBlocks []types.ContentBlock
	partialJSON   map[int]strings.Builder // per-block JSON accumulator for tool_use inputs
}

// NewDeltaAssembler creates an assembler ready to receive events.
func NewDeltaAssembler() *DeltaAssembler {
	return &DeltaAssembler{
		partialJSON: make(map[int]strings.Builder),
	}
}

// Apply processes a single StreamEvent, updating the internal message state.
// Returns true if the message is complete (message_stop received).
func (da *DeltaAssembler) Apply(evt types.StreamEvent) bool {
	switch evt.Type {
	case types.EventMessageStart:
		if evt.Message != nil {
			msg := *evt.Message
			da.message = &msg
			da.contentBlocks = make([]types.ContentBlock, 0, len(msg.Content))
			da.contentBlocks = append(da.contentBlocks, msg.Content...)
		}

	case types.EventContentBlockStart:
		if evt.ContentBlock != nil {
			da.ensureIndex(evt.Index)
			da.contentBlocks[evt.Index] = *evt.ContentBlock
		}

	case types.EventContentBlockDelta:
		if evt.Delta != nil {
			da.applyDelta(evt.Index, evt.Delta)
		}

	case types.EventContentBlockStop:
		// Block is finalized; parse accumulated JSON if any.
		if builder, ok := da.partialJSON[evt.Index]; ok {
			da.ensureIndex(evt.Index)
			block := &da.contentBlocks[evt.Index]
			raw := builder.String()
			if raw != "" {
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &input); err == nil {
					block.Input = input
				} else {
					slog.Warn("failed to parse tool input JSON on block stop",
						"block_index", evt.Index, "error", err)
				}
			}
			delete(da.partialJSON, evt.Index)
		}

	case types.EventMessageDelta:
		if da.message != nil && evt.Delta != nil && evt.Delta.StopReason != "" {
			da.message.StopReason = evt.Delta.StopReason
		}
		if da.message != nil && evt.Usage != nil {
			da.message.Usage = evt.Usage
		}

	case types.EventMessageStop:
		da.finalize()
		return true
	}

	return false
}

// Message returns the assembled message. Only valid after Apply returns true.
func (da *DeltaAssembler) Message() *types.APIMessage {
	return da.message
}

func (da *DeltaAssembler) applyDelta(index int, delta *types.DeltaBlock) {
	da.ensureIndex(index)
	block := &da.contentBlocks[index]

	switch delta.Type {
	case "text_delta":
		block.Text += delta.Text
	case "input_json_delta":
		builder := da.partialJSON[index]
		builder.WriteString(delta.PartialJSON)
		da.partialJSON[index] = builder
	case "thinking_delta":
		block.Thinking += delta.Thinking
	}
}

func (da *DeltaAssembler) ensureIndex(idx int) {
	for len(da.contentBlocks) <= idx {
		da.contentBlocks = append(da.contentBlocks, types.ContentBlock{})
	}
}

func (da *DeltaAssembler) finalize() {
	if da.message == nil {
		return
	}
	// Parse any remaining partial JSON that wasn't closed by content_block_stop.
	for idx, builder := range da.partialJSON {
		if idx < len(da.contentBlocks) {
			raw := builder.String()
			if raw != "" {
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &input); err == nil {
					da.contentBlocks[idx].Input = input
				}
			}
		}
	}
	da.message.Content = da.contentBlocks
}
