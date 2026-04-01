package query

import (
	"unicode"
	"unicode/utf8"

	"github.com/settixx/claude-code-go/internal/types"
)

const (
	charsPerTokenLatin = 4
	charsPerTokenCJK   = 2
)

// EstimateTokens returns a rough token count for the given text.
// It uses ~4 chars/token for Latin script and ~2 chars/token for CJK.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	latinChars, cjkChars := classifyChars(text)

	tokens := latinChars / charsPerTokenLatin
	tokens += cjkChars / charsPerTokenCJK

	if tokens == 0 && utf8.RuneCountInString(text) > 0 {
		return 1
	}
	return tokens
}

// EstimateMessageTokens estimates the token count for a single message,
// including role overhead.
func EstimateMessageTokens(msg types.Message) int {
	const roleOverhead = 4 // role + structural tokens

	total := roleOverhead
	total += estimateContentBlocks(msg.Content)

	if msg.Text != "" {
		total += EstimateTokens(msg.Text)
	}

	if msg.APIMessage != nil {
		total += estimateContentBlocks(msg.APIMessage.Content)
	}

	return total
}

// EstimateConversationTokens sums estimated tokens across all messages.
func EstimateConversationTokens(messages []types.Message) int {
	total := 0
	for i := range messages {
		total += EstimateMessageTokens(messages[i])
	}
	return total
}

func estimateContentBlocks(blocks []types.ContentBlock) int {
	total := 0
	for _, b := range blocks {
		total += estimateBlock(b)
	}
	return total
}

func estimateBlock(b types.ContentBlock) int {
	n := 0
	n += EstimateTokens(b.Text)
	n += EstimateTokens(b.Thinking)
	n += EstimateTokens(b.Name)
	for _, child := range b.Content {
		n += estimateBlock(child)
	}
	return n
}

// classifyChars splits the text's rune count into Latin-ish and CJK buckets.
func classifyChars(text string) (latin, cjk int) {
	for _, r := range text {
		if isCJK(r) {
			cjk++
			continue
		}
		latin++
	}
	return
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hiragana, r)
}
