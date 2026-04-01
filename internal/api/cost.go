package api

import (
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// modelPricing holds per-million-token pricing for a model.
type modelPricing struct {
	InputPerMillion          float64
	OutputPerMillion         float64
	CacheCreationPerMillion  float64
	CacheReadPerMillion      float64
}

// pricingTable maps canonical model IDs to their token pricing (USD per million tokens).
var pricingTable = map[string]modelPricing{
	ModelClaude4Opus: {
		InputPerMillion:         15.0,
		OutputPerMillion:        75.0,
		CacheCreationPerMillion: 18.75,
		CacheReadPerMillion:     1.50,
	},
	ModelClaude4Sonnet: {
		InputPerMillion:         3.0,
		OutputPerMillion:        15.0,
		CacheCreationPerMillion: 3.75,
		CacheReadPerMillion:     0.30,
	},
	ModelClaude35Haiku: {
		InputPerMillion:         0.80,
		OutputPerMillion:        4.0,
		CacheCreationPerMillion: 1.0,
		CacheReadPerMillion:     0.08,
	},
	ModelClaude35Sonnet: {
		InputPerMillion:         3.0,
		OutputPerMillion:        15.0,
		CacheCreationPerMillion: 3.75,
		CacheReadPerMillion:     0.30,
	},
	ModelClaude3Opus: {
		InputPerMillion:         15.0,
		OutputPerMillion:        75.0,
		CacheCreationPerMillion: 18.75,
		CacheReadPerMillion:     1.50,
	},
	ModelClaude3Sonnet: {
		InputPerMillion:         3.0,
		OutputPerMillion:        15.0,
		CacheCreationPerMillion: 3.75,
		CacheReadPerMillion:     0.30,
	},
	ModelClaude3Haiku: {
		InputPerMillion:         0.25,
		OutputPerMillion:        1.25,
		CacheCreationPerMillion: 0.30,
		CacheReadPerMillion:     0.03,
	},
}

// defaultPricing is a conservative fallback for unknown models.
var defaultPricing = modelPricing{
	InputPerMillion:         15.0,
	OutputPerMillion:        75.0,
	CacheCreationPerMillion: 18.75,
	CacheReadPerMillion:     1.50,
}

// EstimateCost computes the USD cost for a single request given model and usage.
func EstimateCost(model string, usage types.Usage) float64 {
	pricing := lookupPricing(model)
	cost := float64(usage.InputTokens) * pricing.InputPerMillion / 1_000_000
	cost += float64(usage.OutputTokens) * pricing.OutputPerMillion / 1_000_000
	cost += float64(usage.CacheCreationInputTokens) * pricing.CacheCreationPerMillion / 1_000_000
	cost += float64(usage.CacheReadInputTokens) * pricing.CacheReadPerMillion / 1_000_000
	return cost
}

func lookupPricing(model string) modelPricing {
	resolved := ResolveModel(model)
	if p, ok := pricingTable[resolved]; ok {
		return p
	}
	return defaultPricing
}

// CostTracker accumulates token usage and cost across multiple API requests.
// It is safe for concurrent use.
type CostTracker struct {
	mu           sync.Mutex
	totalCost    float64
	totalUsage   types.Usage
	requestCount int
}

// NewCostTracker creates a zero-value cost tracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{}
}

// Add records usage from one API response.
func (ct *CostTracker) Add(model string, usage types.Usage) {
	cost := EstimateCost(model, usage)
	ct.mu.Lock()
	ct.totalCost += cost
	ct.totalUsage.InputTokens += usage.InputTokens
	ct.totalUsage.OutputTokens += usage.OutputTokens
	ct.totalUsage.CacheCreationInputTokens += usage.CacheCreationInputTokens
	ct.totalUsage.CacheReadInputTokens += usage.CacheReadInputTokens
	ct.requestCount++
	ct.mu.Unlock()
}

// TotalCost returns the accumulated USD cost.
func (ct *CostTracker) TotalCost() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.totalCost
}

// TotalUsage returns the accumulated token usage.
func (ct *CostTracker) TotalUsage() types.Usage {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.totalUsage
}

// RequestCount returns the number of API requests tracked.
func (ct *CostTracker) RequestCount() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.requestCount
}

// CostSummary is a snapshot of the tracker state.
type CostSummary struct {
	TotalCostUSD float64     `json:"total_cost_usd"`
	TotalUsage   types.Usage `json:"total_usage"`
	RequestCount int         `json:"request_count"`
}

// Summary returns a snapshot of accumulated costs.
func (ct *CostTracker) Summary() CostSummary {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return CostSummary{
		TotalCostUSD: ct.totalCost,
		TotalUsage:   ct.totalUsage,
		RequestCount: ct.requestCount,
	}
}
