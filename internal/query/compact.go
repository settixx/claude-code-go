package query

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	DefaultContextWindowTokens = 200_000
	CompactionThreshold        = 0.80
	RecentMessageKeepCount     = 5
	CompactBoundaryMarker      = "__COMPACT_BOUNDARY__"
)

// CompactionConfig controls when and how compaction runs.
type CompactionConfig struct {
	MaxContextTokens int
	Threshold        float64
	KeepRecent       int
	SummaryModel     string
}

// DefaultCompactionConfig returns production-ready defaults.
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		MaxContextTokens: DefaultContextWindowTokens,
		Threshold:        CompactionThreshold,
		KeepRecent:       RecentMessageKeepCount,
		SummaryModel:     "",
	}
}

func (c CompactionConfig) thresholdTokens() int {
	return int(float64(c.MaxContextTokens) * c.Threshold)
}

// Compactor manages conversation compaction to stay within context limits.
type Compactor struct {
	config  CompactionConfig
	client  interfaces.LLMClient
	tokenFn func(string) int
}

// NewCompactor creates a Compactor. If tokenFn is nil, EstimateTokens is used.
func NewCompactor(cfg CompactionConfig, client interfaces.LLMClient) *Compactor {
	return &Compactor{
		config:  cfg,
		client:  client,
		tokenFn: EstimateTokens,
	}
}

// ShouldCompact returns true when the conversation exceeds the token threshold.
func (c *Compactor) ShouldCompact(messages []types.Message) bool {
	tokens := EstimateConversationTokens(messages)
	threshold := c.config.thresholdTokens()
	return tokens >= threshold
}

// CompactIfNeeded checks token usage and compacts when necessary.
// Returns the (possibly compacted) messages, whether compaction occurred, and any error.
func (c *Compactor) CompactIfNeeded(ctx context.Context, messages []types.Message) ([]types.Message, bool, error) {
	if !c.ShouldCompact(messages) {
		return messages, false, nil
	}

	compacted, err := c.Compact(ctx, messages)
	if err != nil {
		return messages, false, fmt.Errorf("compaction: %w", err)
	}
	return compacted, true, nil
}

// Compact performs conversation compaction by summarizing older messages
// and preserving recent ones verbatim.
func (c *Compactor) Compact(ctx context.Context, messages []types.Message) ([]types.Message, error) {
	if len(messages) <= c.config.KeepRecent {
		return messages, nil
	}

	preTokens := EstimateConversationTokens(messages)
	splitIdx := len(messages) - c.config.KeepRecent
	older := messages[:splitIdx]
	recent := messages[splitIdx:]

	slog.Info("compactor: starting compaction",
		"total_messages", len(messages),
		"summarizing", len(older),
		"keeping", len(recent),
		"pre_tokens", preTokens,
	)

	summary, err := c.summarize(ctx, older)
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}

	compacted := make([]types.Message, 0, 2+len(recent))
	compacted = append(compacted, buildSummaryMessage(summary, len(older)))
	compacted = append(compacted, buildCompactBoundaryMessage())
	compacted = append(compacted, recent...)

	postTokens := EstimateConversationTokens(compacted)
	slog.Info("compactor: compaction complete",
		"pre_tokens", preTokens,
		"post_tokens", postTokens,
		"reduction_pct", fmt.Sprintf("%.1f%%", 100*(1-float64(postTokens)/float64(preTokens))),
	)

	return compacted, nil
}

const summarizationPrompt = `Summarize the following conversation context concisely. Preserve:
- Key decisions made
- Files modified and their purposes
- Outstanding tasks or issues
- Important technical details
Do not include greetings, acknowledgments, or redundant information.`

func (c *Compactor) summarize(ctx context.Context, messages []types.Message) (string, error) {
	prompt := buildSummarizationQuery(messages)

	queryConfig := types.QueryConfig{
		Model:        c.resolveSummaryModel(),
		MaxTokens:    4096,
		SystemPrompt: summarizationPrompt,
	}

	resp, err := c.client.Send(ctx, queryConfig, prompt)
	if err != nil {
		return "", fmt.Errorf("llm send: %w", err)
	}

	return extractTextFromResponse(resp), nil
}

func (c *Compactor) resolveSummaryModel() string {
	if c.config.SummaryModel != "" {
		return c.config.SummaryModel
	}
	return modelIDHaiku45
}

func buildSummarizationQuery(messages []types.Message) []types.Message {
	text := renderMessagesForSummary(messages)
	return []types.Message{{
		Type: types.MsgUser,
		Role: "user",
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: "Please summarize this conversation:\n\n" + text,
		}},
	}}
}

func renderMessagesForSummary(messages []types.Message) string {
	var buf []byte
	for _, m := range messages {
		role := resolveRole(m)
		content := extractMessageText(m)
		if content == "" {
			continue
		}
		buf = append(buf, role...)
		buf = append(buf, ": "...)
		buf = append(buf, content...)
		buf = append(buf, '\n')
	}
	return string(buf)
}

func resolveRole(m types.Message) string {
	if m.Role != "" {
		return m.Role
	}
	switch m.Type {
	case types.MsgUser:
		return "user"
	case types.MsgAssistant:
		return "assistant"
	case types.MsgSystem:
		return "system"
	default:
		return string(m.Type)
	}
}

func extractMessageText(m types.Message) string {
	if m.Text != "" {
		return m.Text
	}
	return contentBlocksText(m.Content)
}

func contentBlocksText(blocks []types.ContentBlock) string {
	var buf []byte
	for _, b := range blocks {
		if b.Text != "" {
			if len(buf) > 0 {
				buf = append(buf, '\n')
			}
			buf = append(buf, b.Text...)
		}
		for _, child := range b.Content {
			if child.Text == "" {
				continue
			}
			if len(buf) > 0 {
				buf = append(buf, '\n')
			}
			buf = append(buf, child.Text...)
		}
	}
	return string(buf)
}

func extractTextFromResponse(resp *types.APIMessage) string {
	if resp == nil {
		return ""
	}
	return contentBlocksText(resp.Content)
}

func buildSummaryMessage(summary string, messagesSummarized int) types.Message {
	return types.Message{
		Type:      types.MsgUser,
		Role:      "user",
		Timestamp: time.Now(),
		Content: []types.ContentBlock{{
			Type: types.ContentText,
			Text: "[Conversation Summary]\n" + summary,
		}},
		IsCompactSummary: true,
		SummarizeMetadata: &types.SummarizeMetadata{
			MessagesSummarized: messagesSummarized,
		},
	}
}

func buildCompactBoundaryMessage() types.Message {
	return types.Message{
		Type:      types.MsgSystem,
		Subtype:   types.SubtypeCompactBoundary,
		Timestamp: time.Now(),
		Text:      CompactBoundaryMarker,
	}
}
