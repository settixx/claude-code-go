package api

import (
	"math"
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		usage   types.Usage
		wantMin float64
		wantMax float64
	}{
		{
			name:  "sonnet basic usage",
			model: ModelClaude4Sonnet,
			usage: types.Usage{
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			wantMin: 17.9,
			wantMax: 18.1,
		},
		{
			name:  "haiku is cheapest known model",
			model: ModelClaude3Haiku,
			usage: types.Usage{
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			wantMin: 1.4,
			wantMax: 1.6,
		},
		{
			name:  "zero usage costs nothing",
			model: ModelClaude4Opus,
			usage: types.Usage{},
			wantMin: 0,
			wantMax: 0.001,
		},
		{
			name:  "cache tokens included",
			model: ModelClaude4Sonnet,
			usage: types.Usage{
				InputTokens:              100_000,
				OutputTokens:             50_000,
				CacheCreationInputTokens: 200_000,
				CacheReadInputTokens:     300_000,
			},
			wantMin: 1.0,
			wantMax: 2.5,
		},
		{
			name:  "unknown model uses default pricing",
			model: "custom-model-v1",
			usage: types.Usage{
				InputTokens:  1000,
				OutputTokens: 1000,
			},
			wantMin: 0.00005,
			wantMax: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateCost(tt.model, tt.usage)
			if cost < tt.wantMin || cost > tt.wantMax {
				t.Errorf("EstimateCost(%q) = %f, want between %f and %f",
					tt.model, cost, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCostTracker(t *testing.T) {
	ct := NewCostTracker()

	if ct.TotalCost() != 0 {
		t.Errorf("initial TotalCost = %f, want 0", ct.TotalCost())
	}
	if ct.RequestCount() != 0 {
		t.Errorf("initial RequestCount = %d, want 0", ct.RequestCount())
	}

	ct.Add(ModelClaude4Sonnet, types.Usage{InputTokens: 100, OutputTokens: 200})
	ct.Add(ModelClaude4Sonnet, types.Usage{InputTokens: 300, OutputTokens: 400})

	if ct.RequestCount() != 2 {
		t.Errorf("RequestCount = %d, want 2", ct.RequestCount())
	}

	usage := ct.TotalUsage()
	if usage.InputTokens != 400 {
		t.Errorf("TotalUsage.InputTokens = %d, want 400", usage.InputTokens)
	}
	if usage.OutputTokens != 600 {
		t.Errorf("TotalUsage.OutputTokens = %d, want 600", usage.OutputTokens)
	}

	if ct.TotalCost() <= 0 {
		t.Error("TotalCost should be positive after adding usage")
	}
}

func TestCostTrackerSummary(t *testing.T) {
	ct := NewCostTracker()
	ct.Add(ModelClaude4Opus, types.Usage{InputTokens: 1000, OutputTokens: 500})

	summary := ct.Summary()
	if summary.RequestCount != 1 {
		t.Errorf("Summary.RequestCount = %d, want 1", summary.RequestCount)
	}
	if summary.TotalUsage.InputTokens != 1000 {
		t.Errorf("Summary.TotalUsage.InputTokens = %d, want 1000", summary.TotalUsage.InputTokens)
	}
	if math.Abs(summary.TotalCostUSD-ct.TotalCost()) > 0.0001 {
		t.Errorf("Summary.TotalCostUSD = %f, want %f", summary.TotalCostUSD, ct.TotalCost())
	}
}
