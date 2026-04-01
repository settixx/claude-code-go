package bridge

import (
	"encoding/json"
	"fmt"
	"time"
)

// BridgeMessageType identifies the kind of bridge protocol message.
type BridgeMessageType string

const (
	MsgPrompt     BridgeMessageType = "prompt"
	MsgResponse   BridgeMessageType = "response"
	MsgToolUse    BridgeMessageType = "tool_use"
	MsgToolResult BridgeMessageType = "tool_result"
	MsgStatus     BridgeMessageType = "status"
	MsgControl    BridgeMessageType = "control"
)

// BridgeMessage is the wire-level envelope for all bridge protocol traffic.
type BridgeMessage struct {
	Type      BridgeMessageType `json:"type"`
	Payload   json.RawMessage   `json:"payload"`
	SessionID string            `json:"session_id"`
	Timestamp time.Time         `json:"timestamp"`
	ID        string            `json:"id,omitempty"`
}

// PromptPayload is the payload for a "prompt" message.
type PromptPayload struct {
	Text        string            `json:"text"`
	Attachments []string          `json:"attachments,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ResponsePayload is the payload for a "response" message.
type ResponsePayload struct {
	Text       string `json:"text"`
	IsPartial  bool   `json:"is_partial,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// ToolUsePayload is the payload for a "tool_use" message.
type ToolUsePayload struct {
	ToolName  string                 `json:"tool_name"`
	ToolUseID string                 `json:"tool_use_id"`
	Input     map[string]interface{} `json:"input"`
}

// ToolResultPayload is the payload for a "tool_result" message.
type ToolResultPayload struct {
	ToolUseID string      `json:"tool_use_id"`
	Data      interface{} `json:"data"`
	IsError   bool        `json:"is_error,omitempty"`
}

// StatusPayload is the payload for a "status" message.
type StatusPayload struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

// ControlPayload is the payload for a "control" message (pause, resume, cancel, etc.).
type ControlPayload struct {
	Action string            `json:"action"`
	Params map[string]string `json:"params,omitempty"`
}

// EncodeBridgeMessage serialises a BridgeMessage to JSON bytes.
func EncodeBridgeMessage(msg BridgeMessage) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("bridge: encode message: %w", err)
	}
	return data, nil
}

// DecodeBridgeMessage deserialises JSON bytes into a BridgeMessage.
func DecodeBridgeMessage(data []byte) (*BridgeMessage, error) {
	var msg BridgeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("bridge: decode message: %w", err)
	}
	return &msg, nil
}

// NewBridgeMessage creates a BridgeMessage with a typed payload.
// The payload value is marshalled to JSON automatically.
func NewBridgeMessage(msgType BridgeMessageType, sessionID string, payload interface{}) (BridgeMessage, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return BridgeMessage{}, fmt.Errorf("bridge: marshal payload: %w", err)
	}
	return BridgeMessage{
		Type:      msgType,
		Payload:   raw,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}, nil
}

// DecodePayload unmarshals the raw payload into the target struct.
func (m *BridgeMessage) DecodePayload(target interface{}) error {
	if m.Payload == nil {
		return fmt.Errorf("bridge: nil payload")
	}
	return json.Unmarshal(m.Payload, target)
}
