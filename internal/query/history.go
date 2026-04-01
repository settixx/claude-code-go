package query

import (
	"sync"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// History is a thread-safe, append-only conversation log that the query
// loop reads from and writes to. It also provides helpers for API message
// preparation, summarisation triggering, and truncation.
type History struct {
	mu       sync.RWMutex
	messages []types.Message
}

// NewHistory creates a History optionally seeded with existing messages.
func NewHistory(initial []types.Message) *History {
	msgs := make([]types.Message, len(initial))
	copy(msgs, initial)
	return &History{messages: msgs}
}

// Append adds one or more messages to the end of the history.
func (h *History) Append(msgs ...types.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msgs...)
}

// All returns a shallow copy of the full message list.
func (h *History) All() []types.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]types.Message, len(h.messages))
	copy(out, h.messages)
	return out
}

// Len returns the current message count.
func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.messages)
}

// Last returns the most recent message, or nil when empty.
func (h *History) Last() *types.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.messages) == 0 {
		return nil
	}
	m := h.messages[len(h.messages)-1]
	return &m
}

// Truncate removes all messages before index, keeping messages[index:].
func (h *History) Truncate(keepFromIndex int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if keepFromIndex <= 0 || keepFromIndex >= len(h.messages) {
		return
	}
	h.messages = append([]types.Message(nil), h.messages[keepFromIndex:]...)
}

// NeedsSummarization reports whether the estimated token count exceeds the
// configured threshold and the history should be compacted.
func NeedsSummarization(tokenCount int, threshold int) bool {
	if threshold <= 0 {
		return false
	}
	return tokenCount >= threshold
}

// PrepareForAPI converts the internal message list into the slice of
// types.Message suitable for the LLM API call. It strips system-only
// messages, ensures alternating user/assistant turns, and normalises
// content blocks.
func (h *History) PrepareForAPI() []types.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return prepareMessages(h.messages)
}

// MessagesAfterCompactBoundary returns only the messages that appear after
// the last compact-boundary system message. If no boundary exists, the
// full list is returned.
func (h *History) MessagesAfterCompactBoundary() []types.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	lastIdx := -1
	for i := len(h.messages) - 1; i >= 0; i-- {
		if isCompactBoundary(h.messages[i]) {
			lastIdx = i
			break
		}
	}
	if lastIdx < 0 {
		out := make([]types.Message, len(h.messages))
		copy(out, h.messages)
		return out
	}

	tail := h.messages[lastIdx:]
	out := make([]types.Message, len(tail))
	copy(out, tail)
	return out
}

func isCompactBoundary(m types.Message) bool {
	return m.Type == types.MsgSystem && m.Subtype == types.SubtypeCompactBoundary
}

// prepareMessages filters and normalises messages for the API wire format.
func prepareMessages(msgs []types.Message) []types.Message {
	var out []types.Message
	for _, m := range msgs {
		if shouldExcludeFromAPI(m) {
			continue
		}
		out = append(out, normalizeForAPI(m))
	}
	return out
}

// shouldExcludeFromAPI returns true for messages that should never reach the API.
func shouldExcludeFromAPI(m types.Message) bool {
	if m.Type == types.MsgSystem {
		return true
	}
	if m.Type == types.MsgProgress {
		return true
	}
	if m.IsVisibleInTranscriptOnly {
		return true
	}
	return false
}

// normalizeForAPI ensures the message has the right shape for the API.
func normalizeForAPI(m types.Message) types.Message {
	if m.Type == types.MsgUser && m.Role == "" {
		m.Role = "user"
	}
	if m.Type == types.MsgAssistant && m.Role == "" {
		m.Role = "assistant"
	}
	return m
}

// NewUserMessage builds a user-role Message with a single text block.
func NewUserMessage(text string) types.Message {
	return types.Message{
		Type:      types.MsgUser,
		Role:      "user",
		Timestamp: time.Now(),
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: text,
		}},
	}
}

// NewToolResultMessage builds a user-role message carrying tool_result blocks.
func NewToolResultMessage(results []types.ContentBlock) types.Message {
	return types.Message{
		Type:      types.MsgUser,
		Role:      "user",
		Timestamp: time.Now(),
		Content:   results,
	}
}

// NewContinueMessage builds a user-role message that prompts the model to
// resume output after hitting the max_tokens limit.
func NewContinueMessage() types.Message {
	return types.Message{
		Type:      types.MsgUser,
		Role:      "user",
		Timestamp: time.Now(),
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: "Please continue from where you left off.",
		}},
	}
}

// NewMultiModalMessage builds a user-role message with arbitrary content
// blocks (text, images, etc.).
func NewMultiModalMessage(blocks []types.ContentBlock) types.Message {
	return types.Message{
		Type:      types.MsgUser,
		Role:      "user",
		Timestamp: time.Now(),
		Content:   blocks,
	}
}

// AssistantMessageFromAPI wraps a raw APIMessage into the conversation
// envelope.
func AssistantMessageFromAPI(api *types.APIMessage) types.Message {
	return types.Message{
		Type:       types.MsgAssistant,
		Role:       "assistant",
		Timestamp:  time.Now(),
		APIMessage: api,
		Content:    api.Content,
	}
}
