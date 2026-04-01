package bridge

import (
	"context"
	"errors"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// HeadlessConfig configures a headless (SDK) runner.
type HeadlessConfig struct {
	Model        string   `json:"model"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Tools        []string `json:"tools,omitempty"`
}

// HeadlessRunner executes Ti Code prompts without a TUI, streaming
// types.Message values back through a channel for programmatic consumption.
type HeadlessRunner struct {
	cfg HeadlessConfig

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewHeadlessRunner creates a HeadlessRunner with the given configuration.
func NewHeadlessRunner(cfg HeadlessConfig) *HeadlessRunner {
	return &HeadlessRunner{cfg: cfg}
}

// Run sends a prompt and returns a channel that streams conversation messages.
// The channel is closed when the conversation finishes or the context is cancelled.
// Callers should range over the returned channel to consume all messages.
func (h *HeadlessRunner) Run(ctx context.Context, prompt string) (<-chan types.Message, error) {
	if prompt == "" {
		return nil, errors.New("headless: empty prompt")
	}

	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return nil, errors.New("headless: already running")
	}
	h.running = true

	ctx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	h.mu.Unlock()

	out := make(chan types.Message, 16)

	go h.execute(ctx, prompt, out)

	return out, nil
}

// Stop cancels the running headless session.
func (h *HeadlessRunner) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cancel != nil {
		h.cancel()
	}
}

// IsRunning reports whether a prompt execution is in progress.
func (h *HeadlessRunner) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

// Config returns a copy of the runner configuration.
func (h *HeadlessRunner) Config() HeadlessConfig {
	return h.cfg
}

// execute is the goroutine that drives a single prompt to completion.
// In the current skeleton it emits a placeholder assistant message;
// the real implementation will plug into the LLM client and tool executor.
func (h *HeadlessRunner) execute(ctx context.Context, prompt string, out chan<- types.Message) {
	defer func() {
		h.mu.Lock()
		h.running = false
		h.cancel = nil
		h.mu.Unlock()
		close(out)
	}()

	userMsg := types.Message{
		Type: types.MsgUser,
		Text: prompt,
		Role: "user",
	}
	if !trySend(ctx, out, userMsg) {
		return
	}

	assistantMsg := types.Message{
		Type: types.MsgAssistant,
		Role: "assistant",
		Text: "[headless placeholder] response to: " + prompt,
	}
	trySend(ctx, out, assistantMsg)
}

func trySend(ctx context.Context, ch chan<- types.Message, msg types.Message) bool {
	select {
	case ch <- msg:
		return true
	case <-ctx.Done():
		return false
	}
}
