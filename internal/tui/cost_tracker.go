package tui

import "fmt"

// CostTracker accumulates token usage and estimated cost from stream events.
type CostTracker struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// TotalTokens returns the combined input + output count.
func (ct *CostTracker) TotalTokens() int {
	return ct.InputTokens + ct.OutputTokens
}

// Add increments token counts and recalculates cost.
func (ct *CostTracker) Add(input, output int) {
	ct.InputTokens += input
	ct.OutputTokens += output
	ct.CostUSD = estimateCost(ct.InputTokens, ct.OutputTokens)
}

// Reset zeroes all fields.
func (ct *CostTracker) Reset() {
	ct.InputTokens = 0
	ct.OutputTokens = 0
	ct.CostUSD = 0
}

// FormatTokensSplit returns a compact "1.2k↓ 450↑" style string.
func (ct *CostTracker) FormatTokensSplit() string {
	return fmt.Sprintf("%s↓ %s↑", compactTokens(ct.InputTokens), compactTokens(ct.OutputTokens))
}

// FormatStatusSegment returns the full "[Tokens: …↓ …↑ | Cost: $…]" string.
func (ct *CostTracker) FormatStatusSegment() string {
	if ct.TotalTokens() == 0 {
		return ""
	}
	return fmt.Sprintf("[Tokens: %s | Cost: %s]", ct.FormatTokensSplit(), FormatCost(ct.CostUSD))
}

func compactTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// estimateCost uses approximate Claude Sonnet pricing as a baseline.
// Input: $3/MTok, Output: $15/MTok.
func estimateCost(input, output int) float64 {
	const inputRate = 3.0 / 1_000_000
	const outputRate = 15.0 / 1_000_000
	return float64(input)*inputRate + float64(output)*outputRate
}
