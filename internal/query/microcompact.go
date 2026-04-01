package query

import (
	"strings"
)

const (
	defaultMaxResultTokens = 10_000
	headRatio              = 0.30
	tailRatio              = 0.30
	omissionMarker         = "\n\n[... middle portion omitted for brevity ...]\n\n"
)

// MicroCompactor truncates or summarizes oversized tool results
// to keep individual messages from blowing up context.
type MicroCompactor struct {
	maxResultTokens int
}

// NewMicroCompactor creates a MicroCompactor. If maxTokens <= 0,
// defaultMaxResultTokens is used.
func NewMicroCompactor(maxTokens int) *MicroCompactor {
	if maxTokens <= 0 {
		maxTokens = defaultMaxResultTokens
	}
	return &MicroCompactor{maxResultTokens: maxTokens}
}

// CompactToolResult truncates a large tool result while preserving
// the head and tail portions.
func (mc *MicroCompactor) CompactToolResult(result string) string {
	if EstimateTokens(result) <= mc.maxResultTokens {
		return result
	}

	if isStructured(result) {
		return mc.compactStructured(result)
	}

	return mc.compactPlain(result)
}

func (mc *MicroCompactor) compactPlain(result string) string {
	runes := []rune(result)
	totalRunes := len(runes)

	headLen := int(float64(totalRunes) * headRatio)
	tailLen := int(float64(totalRunes) * tailRatio)

	if headLen+tailLen >= totalRunes {
		return result
	}

	head := string(runes[:headLen])
	tail := string(runes[totalRunes-tailLen:])
	return head + omissionMarker + tail
}

func (mc *MicroCompactor) compactStructured(result string) string {
	lines := strings.Split(result, "\n")
	totalLines := len(lines)
	if totalLines <= 6 {
		return mc.compactPlain(result)
	}

	headLines := max(3, int(float64(totalLines)*headRatio))
	tailLines := max(3, int(float64(totalLines)*tailRatio))

	if headLines+tailLines >= totalLines {
		return mc.compactPlain(result)
	}

	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[totalLines-tailLines:], "\n")
	return head + omissionMarker + tail
}

func isStructured(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return true
	}
	return strings.Count(s, "\n") > 10
}
