package query

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// Mock LLM client for tests
// ---------------------------------------------------------------------------

type mockLLMClient struct {
	sendFn func(ctx context.Context, config types.QueryConfig, messages []types.Message) (*types.APIMessage, error)
}

func (m *mockLLMClient) Stream(_ context.Context, _ types.QueryConfig, _ []types.Message) (<-chan types.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockLLMClient) Send(ctx context.Context, config types.QueryConfig, messages []types.Message) (*types.APIMessage, error) {
	if m.sendFn != nil {
		return m.sendFn(ctx, config, messages)
	}
	return &types.APIMessage{
		Role: "assistant",
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: "Summary: the user asked about code changes.",
		}},
	}, nil
}

func (m *mockLLMClient) CountTokens(_ context.Context, _ types.QueryConfig, _ []types.Message) (int, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeMessages(n int, textLen int) []types.Message {
	msgs := make([]types.Message, n)
	text := strings.Repeat("x", textLen)
	for i := range msgs {
		role := "user"
		msgType := types.MsgUser
		if i%2 == 1 {
			role = "assistant"
			msgType = types.MsgAssistant
		}
		msgs[i] = types.Message{
			Type:      msgType,
			Role:      role,
			Timestamp: time.Now(),
			Content: []types.ContentBlock{{
				Type: types.ContentText,
				Text: text,
			}},
		}
	}
	return msgs
}

// ---------------------------------------------------------------------------
// ShouldCompact tests
// ---------------------------------------------------------------------------

func TestShouldCompact_UnderThreshold(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 200_000,
		Threshold:        0.80,
		KeepRecent:       5,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	msgs := makeMessages(3, 20)
	if c.ShouldCompact(msgs) {
		t.Error("should not compact: messages are well under threshold")
	}
}

func TestShouldCompact_OverThreshold(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       2,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	// Threshold = 50 tokens. Each message with 200-char text ≈ 54 tokens.
	msgs := makeMessages(5, 200)
	if !c.ShouldCompact(msgs) {
		tokens := EstimateConversationTokens(msgs)
		t.Errorf("should compact: tokens=%d, threshold=%d", tokens, cfg.thresholdTokens())
	}
}

// ---------------------------------------------------------------------------
// Compact tests
// ---------------------------------------------------------------------------

func TestCompact_PreservesRecentMessages(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       3,
	}
	client := &mockLLMClient{}
	c := NewCompactor(cfg, client)

	msgs := makeMessages(10, 100)
	result, err := c.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: 1 summary + 1 boundary + 3 recent = 5
	if len(result) != 5 {
		t.Fatalf("want 5 messages, got %d", len(result))
	}

	// Last 3 should be the original recent messages
	for i := 0; i < 3; i++ {
		original := msgs[7+i]
		got := result[2+i]
		if got.Role != original.Role {
			t.Errorf("recent[%d]: role mismatch: %s vs %s", i, got.Role, original.Role)
		}
	}
}

func TestCompact_InsertsBoundaryMarker(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       2,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	msgs := makeMessages(8, 100)
	result, err := c.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boundary := result[1]
	if boundary.Type != types.MsgSystem {
		t.Errorf("boundary type: want %s, got %s", types.MsgSystem, boundary.Type)
	}
	if boundary.Subtype != types.SubtypeCompactBoundary {
		t.Errorf("boundary subtype: want %s, got %s", types.SubtypeCompactBoundary, boundary.Subtype)
	}
	if boundary.Text != CompactBoundaryMarker {
		t.Errorf("boundary text: want %q, got %q", CompactBoundaryMarker, boundary.Text)
	}
}

func TestCompact_SummaryMessage(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       2,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	msgs := makeMessages(6, 100)
	result, err := c.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summary := result[0]
	if !summary.IsCompactSummary {
		t.Error("first message should be marked as compact summary")
	}
	if summary.SummarizeMetadata == nil {
		t.Fatal("summary should have SummarizeMetadata")
	}
	if summary.SummarizeMetadata.MessagesSummarized != 4 {
		t.Errorf("messages summarized: want 4, got %d", summary.SummarizeMetadata.MessagesSummarized)
	}
	if !strings.Contains(contentBlocksText(summary.Content), "[Conversation Summary]") {
		t.Error("summary should contain [Conversation Summary] header")
	}
}

func TestCompact_TooFewMessages(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       5,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	msgs := makeMessages(3, 100)
	result, err := c.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("should return original messages when count <= KeepRecent, got %d", len(result))
	}
}

func TestCompact_LLMError(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.50,
		KeepRecent:       2,
	}
	client := &mockLLMClient{
		sendFn: func(_ context.Context, _ types.QueryConfig, _ []types.Message) (*types.APIMessage, error) {
			return nil, fmt.Errorf("api timeout")
		},
	}
	c := NewCompactor(cfg, client)

	msgs := makeMessages(6, 100)
	_, err := c.Compact(context.Background(), msgs)
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
	if !strings.Contains(err.Error(), "api timeout") {
		t.Errorf("error should contain 'api timeout', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CompactIfNeeded tests
// ---------------------------------------------------------------------------

func TestCompactIfNeeded_NoCompaction(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 200_000,
		Threshold:        0.80,
		KeepRecent:       5,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	msgs := makeMessages(3, 20)
	result, compacted, err := c.CompactIfNeeded(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compacted {
		t.Error("should not have compacted")
	}
	if len(result) != len(msgs) {
		t.Errorf("messages unchanged: want %d, got %d", len(msgs), len(result))
	}
}

func TestCompactIfNeeded_TriggersCompaction(t *testing.T) {
	cfg := CompactionConfig{
		MaxContextTokens: 100,
		Threshold:        0.10,
		KeepRecent:       2,
	}
	c := NewCompactor(cfg, &mockLLMClient{})

	msgs := makeMessages(10, 200)
	result, compacted, err := c.CompactIfNeeded(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !compacted {
		t.Error("should have compacted")
	}
	if len(result) >= len(msgs) {
		t.Errorf("compacted len (%d) should be less than original (%d)", len(result), len(msgs))
	}
}

// ---------------------------------------------------------------------------
// DefaultCompactionConfig test
// ---------------------------------------------------------------------------

func TestDefaultCompactionConfig(t *testing.T) {
	cfg := DefaultCompactionConfig()
	if cfg.MaxContextTokens != DefaultContextWindowTokens {
		t.Errorf("MaxContextTokens: want %d, got %d", DefaultContextWindowTokens, cfg.MaxContextTokens)
	}
	if cfg.Threshold != CompactionThreshold {
		t.Errorf("Threshold: want %f, got %f", CompactionThreshold, cfg.Threshold)
	}
	if cfg.KeepRecent != RecentMessageKeepCount {
		t.Errorf("KeepRecent: want %d, got %d", RecentMessageKeepCount, cfg.KeepRecent)
	}
}

// ---------------------------------------------------------------------------
// MicroCompactor tests
// ---------------------------------------------------------------------------

func TestMicroCompact_ShortResult(t *testing.T) {
	mc := NewMicroCompactor(10_000)
	input := "short result"
	got := mc.CompactToolResult(input)
	if got != input {
		t.Errorf("short result should be unchanged, got %q", got)
	}
}

func TestMicroCompact_LongResult(t *testing.T) {
	mc := NewMicroCompactor(100)

	input := strings.Repeat("word ", 2000)
	got := mc.CompactToolResult(input)

	if !strings.Contains(got, "[... middle portion omitted for brevity ...]") {
		t.Error("truncated result should contain omission marker")
	}
	if len(got) >= len(input) {
		t.Errorf("truncated result (%d) should be shorter than input (%d)", len(got), len(input))
	}
}

func TestMicroCompact_StructuredJSON(t *testing.T) {
	mc := NewMicroCompactor(50)

	lines := make([]string, 50)
	lines[0] = "{"
	for i := 1; i < 49; i++ {
		lines[i] = fmt.Sprintf(`  "key%d": "value%d",`, i, i)
	}
	lines[49] = "}"
	input := strings.Join(lines, "\n")

	got := mc.CompactToolResult(input)
	if !strings.Contains(got, "[... middle portion omitted for brevity ...]") {
		t.Error("structured result should contain omission marker")
	}
}

func TestMicroCompact_DefaultTokenLimit(t *testing.T) {
	mc := NewMicroCompactor(0)
	if mc.maxResultTokens != defaultMaxResultTokens {
		t.Errorf("default: want %d, got %d", defaultMaxResultTokens, mc.maxResultTokens)
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestContentBlocksText(t *testing.T) {
	blocks := []types.ContentBlock{
		{Type: types.ContentText, Text: "hello"},
		{Type: types.ContentText, Text: "world"},
	}
	got := contentBlocksText(blocks)
	if got != "hello\nworld" {
		t.Errorf("want %q, got %q", "hello\nworld", got)
	}
}

func TestResolveRole(t *testing.T) {
	tests := []struct {
		msg  types.Message
		want string
	}{
		{types.Message{Role: "user"}, "user"},
		{types.Message{Role: "assistant"}, "assistant"},
		{types.Message{Type: types.MsgUser}, "user"},
		{types.Message{Type: types.MsgAssistant}, "assistant"},
		{types.Message{Type: types.MsgSystem}, "system"},
	}
	for _, tt := range tests {
		got := resolveRole(tt.msg)
		if got != tt.want {
			t.Errorf("resolveRole(%v): want %q, got %q", tt.msg.Type, tt.want, got)
		}
	}
}
