package query

import (
	"log/slog"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// Budget tracks cumulative token consumption and enforces spending limits
// across a conversation session. It is safe for concurrent use.
type Budget struct {
	mu sync.Mutex

	totalInput  int
	totalOutput int
	totalCache  int

	maxTokens int
	maxUSD    float64

	inputPrice  float64
	outputPrice float64
}

// BudgetConfig holds the limits and pricing used to initialise a Budget.
type BudgetConfig struct {
	MaxTokens   int
	MaxUSD      float64
	InputPrice  float64
	OutputPrice float64
}

// NewBudget creates a Budget with the given limits.
// Pricing is expressed as USD per token (e.g. $3/1M tokens → 0.000003).
func NewBudget(cfg BudgetConfig) *Budget {
	return &Budget{
		maxTokens:   cfg.MaxTokens,
		maxUSD:      cfg.MaxUSD,
		inputPrice:  cfg.InputPrice,
		outputPrice: cfg.OutputPrice,
	}
}

// RecordUsage adds a single API response's token counts to the running totals.
func (b *Budget) RecordUsage(u types.Usage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.totalInput += u.InputTokens
	b.totalOutput += u.OutputTokens
	b.totalCache += u.CacheCreationInputTokens + u.CacheReadInputTokens

	slog.Debug("budget: recorded usage",
		"input", u.InputTokens,
		"output", u.OutputTokens,
		"total_input", b.totalInput,
		"total_output", b.totalOutput,
	)
}

// CanContinue reports whether the budget still has room for more queries.
func (b *Budget) CanContinue() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.exceededLocked()
}

// Exhausted reports whether any hard limit has been reached.
func (b *Budget) Exhausted() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exceededLocked()
}

// Remaining returns the headroom left in both token and USD dimensions.
// A non-positive value means the corresponding limit was already exceeded.
func (b *Budget) Remaining() (tokens int, usd float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	tokens = b.remainingTokensLocked()
	usd = b.remainingUSDLocked()
	return
}

// TotalTokens returns the aggregate token count (input + output + cache).
func (b *Budget) TotalTokens() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.totalInput + b.totalOutput + b.totalCache
}

// TotalCostUSD returns the estimated spend based on token counts and pricing.
func (b *Budget) TotalCostUSD() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.costLocked()
}

// Usage returns a snapshot of the cumulative usage counters.
func (b *Budget) Usage() types.Usage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return types.Usage{
		InputTokens:  b.totalInput,
		OutputTokens: b.totalOutput,
	}
}

func (b *Budget) exceededLocked() bool {
	if b.maxTokens > 0 && b.remainingTokensLocked() <= 0 {
		return true
	}
	if b.maxUSD > 0 && b.remainingUSDLocked() <= 0 {
		return true
	}
	return false
}

func (b *Budget) remainingTokensLocked() int {
	if b.maxTokens <= 0 {
		return 1 // unlimited
	}
	return b.maxTokens - (b.totalInput + b.totalOutput + b.totalCache)
}

func (b *Budget) remainingUSDLocked() float64 {
	if b.maxUSD <= 0 {
		return 1.0 // unlimited
	}
	return b.maxUSD - b.costLocked()
}

func (b *Budget) costLocked() float64 {
	return float64(b.totalInput)*b.inputPrice + float64(b.totalOutput)*b.outputPrice
}
