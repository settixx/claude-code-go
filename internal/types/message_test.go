package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageCreation(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		wantRole string
	}{
		{"user message", MsgUser, "user"},
		{"assistant message", MsgAssistant, "assistant"},
		{"system message", MsgSystem, ""},
		{"attachment message", MsgAttachment, ""},
		{"progress message", MsgProgress, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := Message{
				Type:      tt.msgType,
				UUID:      "test-uuid",
				Timestamp: time.Now(),
				Role:      tt.wantRole,
			}
			if msg.Type != tt.msgType {
				t.Errorf("Type = %q, want %q", msg.Type, tt.msgType)
			}
			if msg.Role != tt.wantRole {
				t.Errorf("Role = %q, want %q", msg.Role, tt.wantRole)
			}
		})
	}
}

func TestContentBlockJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
	}{
		{
			name:  "text block",
			block: ContentBlock{Type: ContentText, Text: "hello world"},
		},
		{
			name: "tool_use block",
			block: ContentBlock{
				Type: ContentToolUse,
				ID:   "toolu_123",
				Name: "Bash",
				Input: map[string]interface{}{
					"command": "echo hi",
				},
			},
		},
		{
			name: "tool_result block",
			block: ContentBlock{
				Type:      ContentToolResult,
				ToolUseID: "toolu_123",
				Content: []ContentBlock{
					{Type: ContentText, Text: "hi\n"},
				},
			},
		},
		{
			name: "image block",
			block: ContentBlock{
				Type: ContentImage,
				Source: &ImageSource{
					Type:      "base64",
					MediaType: "image/png",
					Data:      "iVBOR",
				},
			},
		},
		{
			name:  "thinking block",
			block: ContentBlock{Type: ContentThinking, Thinking: "Let me think..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.block)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got ContentBlock
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.Type != tt.block.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.block.Type)
			}
			if got.Text != tt.block.Text {
				t.Errorf("Text = %q, want %q", got.Text, tt.block.Text)
			}
			if got.ID != tt.block.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.block.ID)
			}
			if got.Thinking != tt.block.Thinking {
				t.Errorf("Thinking = %q, want %q", got.Thinking, tt.block.Thinking)
			}
		})
	}
}

func TestStopReasonConstants(t *testing.T) {
	tests := []struct {
		reason StopReason
		want   string
	}{
		{StopEndTurn, "end_turn"},
		{StopMaxTokens, "max_tokens"},
		{StopToolUse, "tool_use"},
	}

	for _, tt := range tests {
		if string(tt.reason) != tt.want {
			t.Errorf("StopReason %v = %q, want %q", tt.reason, string(tt.reason), tt.want)
		}
	}
}

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		mt   MessageType
		want string
	}{
		{MsgUser, "user"},
		{MsgAssistant, "assistant"},
		{MsgSystem, "system"},
		{MsgAttachment, "attachment"},
		{MsgProgress, "progress"},
	}

	for _, tt := range tests {
		if string(tt.mt) != tt.want {
			t.Errorf("MessageType = %q, want %q", string(tt.mt), tt.want)
		}
	}
}

func TestAPIMessageJSONRoundTrip(t *testing.T) {
	msg := APIMessage{
		ID:         "msg_001",
		Type:       "message",
		Role:       "assistant",
		Content:    []ContentBlock{{Type: ContentText, Text: "Hello"}},
		Model:      "claude-4-sonnet-20260301",
		StopReason: StopEndTurn,
		Usage:      &Usage{InputTokens: 10, OutputTokens: 20},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got APIMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.ID != msg.ID {
		t.Errorf("ID = %q, want %q", got.ID, msg.ID)
	}
	if got.StopReason != StopEndTurn {
		t.Errorf("StopReason = %q, want %q", got.StopReason, StopEndTurn)
	}
	if got.Usage.InputTokens != 10 || got.Usage.OutputTokens != 20 {
		t.Errorf("Usage = %+v, want input=10, output=20", got.Usage)
	}
}
