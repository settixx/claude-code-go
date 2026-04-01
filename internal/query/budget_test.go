package query

import (
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

func TestBudgetCanContinueInitially(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 1000})
	if !b.CanContinue() {
		t.Error("fresh budget should allow continuation")
	}
}

func TestBudgetTokenTracking(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 1000})

	b.RecordUsage(types.Usage{InputTokens: 200, OutputTokens: 100})
	b.RecordUsage(types.Usage{InputTokens: 300, OutputTokens: 150})

	total := b.TotalTokens()
	if total != 750 {
		t.Errorf("TotalTokens = %d, want 750", total)
	}

	usage := b.Usage()
	if usage.InputTokens != 500 {
		t.Errorf("Usage.InputTokens = %d, want 500", usage.InputTokens)
	}
	if usage.OutputTokens != 250 {
		t.Errorf("Usage.OutputTokens = %d, want 250", usage.OutputTokens)
	}
}

func TestBudgetExhaustedByTokens(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 100})

	b.RecordUsage(types.Usage{InputTokens: 60, OutputTokens: 50})

	if b.CanContinue() {
		t.Error("budget should be exhausted after exceeding MaxTokens")
	}
	if !b.Exhausted() {
		t.Error("Exhausted() should be true")
	}
}

func TestBudgetExhaustedByUSD(t *testing.T) {
	b := NewBudget(BudgetConfig{
		MaxUSD:      0.01,
		InputPrice:  0.000003,
		OutputPrice: 0.000015,
	})

	b.RecordUsage(types.Usage{InputTokens: 1000, OutputTokens: 1000})

	if b.CanContinue() {
		t.Error("budget should be exhausted after exceeding MaxUSD")
	}
}

func TestBudgetUnlimitedWhenZeroLimits(t *testing.T) {
	b := NewBudget(BudgetConfig{})

	b.RecordUsage(types.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000})

	if !b.CanContinue() {
		t.Error("unlimited budget should always allow continuation")
	}
	if b.Exhausted() {
		t.Error("unlimited budget should never be exhausted")
	}
}

func TestBudgetRemaining(t *testing.T) {
	b := NewBudget(BudgetConfig{
		MaxTokens:   1000,
		MaxUSD:      1.0,
		InputPrice:  0.000003,
		OutputPrice: 0.000015,
	})

	b.RecordUsage(types.Usage{InputTokens: 200, OutputTokens: 100})

	tokens, usd := b.Remaining()
	if tokens != 700 {
		t.Errorf("remaining tokens = %d, want 700", tokens)
	}
	if usd <= 0 || usd >= 1.0 {
		t.Errorf("remaining USD = %f, want between 0 and 1.0", usd)
	}
}

func TestBudgetTotalCostUSD(t *testing.T) {
	b := NewBudget(BudgetConfig{
		InputPrice:  0.000003,
		OutputPrice: 0.000015,
	})

	b.RecordUsage(types.Usage{InputTokens: 1_000_000, OutputTokens: 100_000})

	cost := b.TotalCostUSD()
	wantMin := 4.4
	wantMax := 4.6
	if cost < wantMin || cost > wantMax {
		t.Errorf("TotalCostUSD = %f, want between %f and %f", cost, wantMin, wantMax)
	}
}

func TestBudgetCacheTokensCounted(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 500})

	b.RecordUsage(types.Usage{
		InputTokens:              100,
		OutputTokens:             100,
		CacheCreationInputTokens: 200,
		CacheReadInputTokens:     200,
	})

	total := b.TotalTokens()
	if total != 600 {
		t.Errorf("TotalTokens = %d, want 600 (includes cache)", total)
	}

	if b.CanContinue() {
		t.Error("should be exhausted: total 600 > limit 500")
	}
}
