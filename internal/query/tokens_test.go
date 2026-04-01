package query

import (
	"strings"
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

func TestEstimateTokens_Empty(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("empty string: want 0, got %d", got)
	}
}

func TestEstimateTokens_EnglishText(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog"
	tokens := EstimateTokens(text)

	// ~44 chars / 4 ≈ 11 tokens
	if tokens < 8 || tokens > 15 {
		t.Errorf("English text: want 8-15, got %d (len=%d)", tokens, len(text))
	}
}

func TestEstimateTokens_CJKText(t *testing.T) {
	text := "这是一个测试句子用来检查中文"
	tokens := EstimateTokens(text)

	// 14 CJK chars / 2 = 7 tokens
	if tokens < 5 || tokens > 10 {
		t.Errorf("CJK text: want 5-10, got %d", tokens)
	}
}

func TestEstimateTokens_MixedContent(t *testing.T) {
	text := "Hello 你好 World 世界"
	tokens := EstimateTokens(text)

	if tokens < 3 || tokens > 10 {
		t.Errorf("mixed content: want 3-10, got %d", tokens)
	}
}

func TestEstimateTokens_SingleChar(t *testing.T) {
	if got := EstimateTokens("a"); got != 1 {
		t.Errorf("single char: want 1, got %d", got)
	}
}

func TestEstimateTokens_SingleCJKChar(t *testing.T) {
	if got := EstimateTokens("你"); got != 1 {
		t.Errorf("single CJK char: want 1, got %d", got)
	}
}

func TestEstimateTokens_LongText(t *testing.T) {
	text := strings.Repeat("word ", 1000)
	tokens := EstimateTokens(text)
	// 5000 chars / 4 = 1250
	if tokens < 1000 || tokens > 1500 {
		t.Errorf("long text: want 1000-1500, got %d", tokens)
	}
}

func TestEstimateMessageTokens_UserText(t *testing.T) {
	msg := types.Message{
		Type: types.MsgUser,
		Role: "user",
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: "Hello, can you help me with this code?",
		}},
	}

	tokens := EstimateMessageTokens(msg)
	if tokens <= 4 {
		t.Errorf("user message should have more than role overhead, got %d", tokens)
	}
}

func TestEstimateMessageTokens_EmptyContent(t *testing.T) {
	msg := types.Message{
		Type: types.MsgUser,
		Role: "user",
	}

	tokens := EstimateMessageTokens(msg)
	// Should be just role overhead (4)
	if tokens != 4 {
		t.Errorf("empty message: want 4 (overhead), got %d", tokens)
	}
}

func TestEstimateConversationTokens(t *testing.T) {
	messages := []types.Message{
		{
			Type: types.MsgUser,
			Role: "user",
			Content: []types.ContentBlock{{
				Type: types.ContentText,
				Text: "Hello",
			}},
		},
		{
			Type: types.MsgAssistant,
			Role: "assistant",
			Content: []types.ContentBlock{{
				Type: types.ContentText,
				Text: "Hi there! How can I help?",
			}},
		},
	}

	total := EstimateConversationTokens(messages)
	sum := EstimateMessageTokens(messages[0]) + EstimateMessageTokens(messages[1])
	if total != sum {
		t.Errorf("conversation tokens (%d) != sum of message tokens (%d)", total, sum)
	}
}

func TestEstimateConversationTokens_Empty(t *testing.T) {
	if got := EstimateConversationTokens(nil); got != 0 {
		t.Errorf("nil messages: want 0, got %d", got)
	}
}

func TestEstimateMessageTokens_WithAPIMessage(t *testing.T) {
	msg := types.Message{
		Type: types.MsgAssistant,
		Role: "assistant",
		APIMessage: &types.APIMessage{
			Content: []types.ContentBlock{{
				Type: types.ContentText,
				Text: "This is the API response content",
			}},
		},
	}

	tokens := EstimateMessageTokens(msg)
	if tokens <= 4 {
		t.Errorf("message with APIMessage should have content tokens, got %d", tokens)
	}
}
