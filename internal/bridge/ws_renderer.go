package bridge

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/settixx/claude-code-go/internal/types"
)

// wsEnvelope is the JSON envelope sent over WebSocket for every outbound event.
type wsEnvelope struct {
	Type    string      `json:"type"`
	Message interface{} `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Text    string      `json:"text,omitempty"`
}

// WebSocketRenderer implements interfaces.Renderer by serialising
// every message/error as JSON and writing it to a WebSocket connection.
type WebSocketRenderer struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// NewWebSocketRenderer wraps an established WebSocket connection.
func NewWebSocketRenderer(conn *websocket.Conn) *WebSocketRenderer {
	return &WebSocketRenderer{conn: conn}
}

func (r *WebSocketRenderer) RenderMessage(msg types.Message) {
	r.sendJSON(wsEnvelope{Type: "message", Message: msg})
}

func (r *WebSocketRenderer) RenderError(err error) {
	r.sendJSON(wsEnvelope{Type: "error", Error: err.Error()})
}

func (r *WebSocketRenderer) RenderSpinner(text string) {
	r.sendJSON(wsEnvelope{Type: "spinner", Text: text})
}

func (r *WebSocketRenderer) StopSpinner() {
	r.sendJSON(wsEnvelope{Type: "spinner_stop"})
}

func (r *WebSocketRenderer) sendJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.conn.WriteMessage(websocket.TextMessage, data)
}
