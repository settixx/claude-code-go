package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/query"
)

const (
	pingInterval = 30 * time.Second
	pongTimeout  = 60 * time.Second
)

// HeadlessConfig configures a headless (SDK) runner.
type HeadlessConfig struct {
	Model        string   `json:"model"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	MaxTurns     int      `json:"max_turns,omitempty"`
	Tools        []string `json:"tools,omitempty"`
	CWD          string   `json:"cwd,omitempty"`

	LLMClient         interfaces.LLMClient
	ToolExecutor      interfaces.ToolExecutor
	StateStore        interfaces.StateStore
	SessionStorage    interfaces.SessionStorage
	PermissionChecker interfaces.PermissionChecker
}

// HeadlessRunner executes Ti Code prompts without a TUI, streaming
// results back through a WebSocket connection for programmatic consumption.
type HeadlessRunner struct {
	cfg HeadlessConfig

	mu     sync.Mutex
	engine *query.Engine
	cancel context.CancelFunc
}

// NewHeadlessRunner creates a HeadlessRunner with the given configuration.
func NewHeadlessRunner(cfg HeadlessConfig) *HeadlessRunner {
	return &HeadlessRunner{cfg: cfg}
}

// Config returns a copy of the runner configuration.
func (h *HeadlessRunner) Config() HeadlessConfig {
	return h.cfg
}

// createEngine builds a fresh query.Engine wired to the given renderer.
func (h *HeadlessRunner) createEngine(renderer interfaces.Renderer) *query.Engine {
	maxTokens := h.cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16384
	}

	return query.NewEngine(query.EngineConfig{
		LLMClient:         h.cfg.LLMClient,
		ToolExecutor:      h.cfg.ToolExecutor,
		StateStore:        h.cfg.StateStore,
		SessionStorage:    h.cfg.SessionStorage,
		Renderer:          renderer,
		PermissionChecker: h.cfg.PermissionChecker,
		SystemPrompt:      h.cfg.SystemPrompt,
		Model:             h.cfg.Model,
		MaxTokens:         maxTokens,
		MaxTurns:          h.cfg.MaxTurns,
		CWD:               h.cfg.CWD,
	})
}

// execute runs a single prompt through the query engine, streaming
// results to the WebSocket-backed renderer.
func (h *HeadlessRunner) execute(ctx context.Context, input string, renderer interfaces.Renderer) error {
	if input == "" {
		return errors.New("headless: empty prompt")
	}

	h.mu.Lock()
	if h.engine == nil {
		h.engine = h.createEngine(renderer)
	}
	h.mu.Unlock()

	return h.engine.Run(ctx, input)
}

// resetEngine discards the current engine so the next execute() starts fresh.
func (h *HeadlessRunner) resetEngine() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.engine = nil
}

// Stop cancels the running headless session.
func (h *HeadlessRunner) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cancel != nil {
		h.cancel()
	}
}

// --- SDK WebSocket server -------------------------------------------------

// sdkMessage is the JSON envelope received from SDK clients.
type sdkMessage struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Action string `json:"action,omitempty"`
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// ServeSDK is an http.Handler that upgrades to WebSocket and serves the
// SDK protocol: clients send JSON commands, the server streams responses.
func (h *HeadlessRunner) ServeSDK(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("headless: websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	h.mu.Lock()
	h.cancel = cancel
	h.mu.Unlock()

	go h.startKeepalive(ctx, conn)

	renderer := NewWebSocketRenderer(conn)
	h.readSDKMessages(ctx, conn, renderer, cancel)
}

// readSDKMessages is the main read loop for SDK mode.
func (h *HeadlessRunner) readSDKMessages(
	ctx context.Context,
	conn *websocket.Conn,
	renderer *WebSocketRenderer,
	cancel context.CancelFunc,
) {
	conn.SetReadDeadline(time.Now().Add(pongTimeout))

	var execCancel context.CancelFunc
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Debug("headless: read error", "error", err)
			}
			return
		}

		var msg sdkMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			renderer.RenderError(fmt.Errorf("invalid JSON: %w", err))
			continue
		}

		switch msg.Type {
		case "query":
			execCancel = h.handleQuery(ctx, msg.Text, renderer, conn)

		case "cancel":
			if execCancel != nil {
				execCancel()
				execCancel = nil
			}
			renderer.sendJSON(wsEnvelope{Type: "cancelled"})

		case "session":
			h.handleSessionAction(msg.Action, renderer)

		default:
			renderer.RenderError(fmt.Errorf("unknown message type: %q", msg.Type))
		}

		if ctx.Err() != nil {
			return
		}
	}
}

// handleQuery starts a query execution in a background goroutine,
// returning the cancel func so the caller can abort it.
func (h *HeadlessRunner) handleQuery(
	ctx context.Context,
	text string,
	renderer *WebSocketRenderer,
	conn *websocket.Conn,
) context.CancelFunc {
	execCtx, execCancel := context.WithCancel(ctx)

	go func() {
		renderer.sendJSON(wsEnvelope{Type: "status", Text: "running"})

		if err := h.execute(execCtx, text, renderer); err != nil {
			renderer.RenderError(err)
		}

		renderer.sendJSON(wsEnvelope{Type: "status", Text: "done"})
	}()

	return execCancel
}

// handleSessionAction processes session-level control messages.
func (h *HeadlessRunner) handleSessionAction(action string, renderer *WebSocketRenderer) {
	switch action {
	case "new":
		h.resetEngine()
		renderer.sendJSON(wsEnvelope{Type: "session", Text: "new session started"})
	default:
		renderer.RenderError(fmt.Errorf("unknown session action: %q", action))
	}
}

// startKeepalive sends periodic WebSocket pings and enforces the pong
// deadline so stale connections are detected promptly.
func (h *HeadlessRunner) startKeepalive(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deadline := time.Now().Add(5 * time.Second)
			if err := conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				slog.Debug("headless: ping failed", "error", err)
				return
			}
		}
	}
}
