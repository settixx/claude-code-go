package tui

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// GitBranchCache — extra coverage
// ---------------------------------------------------------------------------

func TestGitBranchCache_NewCustomTTL(t *testing.T) {
	cache := NewGitBranchCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("NewGitBranchCache should not return nil")
	}
	if cache.ttl != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", cache.ttl)
	}
}

func TestGitBranchCache_ShortTTL_Refetches(t *testing.T) {
	cache := NewGitBranchCache(1 * time.Nanosecond)

	_ = cache.Branch()
	time.Sleep(2 * time.Millisecond)

	// Second call should re-fetch because TTL is essentially zero
	_ = cache.Branch()
}

func TestGitBranchCache_InvalidateResetsTime(t *testing.T) {
	cache := NewGitBranchCache(1 * time.Hour)
	_ = cache.Branch()
	cache.Invalidate()

	cache.mu.Lock()
	isZero := cache.fetchedAt.IsZero()
	cache.mu.Unlock()

	if !isZero {
		t.Error("fetchedAt should be zero after Invalidate")
	}
}

// ---------------------------------------------------------------------------
// BuddyWidget — extra edge cases
// ---------------------------------------------------------------------------

func TestBuddyWidget_NarrowWidth(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("X")
	bw.SetWidth(5)

	view := bw.View()
	if view == "" {
		t.Error("should still render something")
	}
}

func TestBuddyWidget_DefaultWidthUsed(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("X")
	// No SetWidth — default 40 should be used

	view := bw.View()
	if view == "" {
		t.Error("should render with default width")
	}
}

func TestBuddyWidget_TextWrapping(t *testing.T) {
	bw := NewBuddyWidget()
	bw.SetVisible(true)
	bw.SetFrame("O")
	bw.SetText("this is a long text that should be wrapped properly into multiple lines")
	bw.SetWidth(30)

	view := bw.View()
	if view == "" {
		t.Error("should render with text")
	}
	if bw.Height() < 2 {
		t.Errorf("height with wrapping text should be > 1, got %d", bw.Height())
	}
}

// ---------------------------------------------------------------------------
// CostTracker — extra edge cases
// ---------------------------------------------------------------------------

func TestCostTracker_CostCalculation(t *testing.T) {
	ct := &CostTracker{}
	ct.Add(1_000_000, 0)

	// Input: $3/MTok → 1M tokens = $3.00
	if ct.CostUSD < 2.99 || ct.CostUSD > 3.01 {
		t.Errorf("CostUSD for 1M input tokens = %f, want ~3.0", ct.CostUSD)
	}
}

func TestCostTracker_OutputCostCalculation(t *testing.T) {
	ct := &CostTracker{}
	ct.Add(0, 1_000_000)

	// Output: $15/MTok → 1M tokens = $15.00
	if ct.CostUSD < 14.99 || ct.CostUSD > 15.01 {
		t.Errorf("CostUSD for 1M output tokens = %f, want ~15.0", ct.CostUSD)
	}
}

// ---------------------------------------------------------------------------
// Wrap — edge cases
// ---------------------------------------------------------------------------

func TestWrap_EmptyString(t *testing.T) {
	wrapped := Wrap("", 40)
	if wrapped != "" {
		t.Errorf("Wrap('') = %q, want empty", wrapped)
	}
}

func TestWrap_ShortString(t *testing.T) {
	wrapped := Wrap("hi", 40)
	if strings.Contains(wrapped, "\n") {
		t.Error("short string should not be wrapped")
	}
}

func TestWrap_SingleLongWord(t *testing.T) {
	long := strings.Repeat("x", 100)
	wrapped := Wrap(long, 40)
	if wrapped == "" {
		t.Error("should return something for long word")
	}
}
