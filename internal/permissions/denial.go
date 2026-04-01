package permissions

import "sync"

const defaultFallbackThreshold = 3

// DenialTracker records consecutive permission denials per tool. After a
// configurable number of consecutive denials, the system should fall back to
// prompting the user instead of silently denying.
type DenialTracker struct {
	mu        sync.Mutex
	counts    map[string]int
	threshold int
}

// NewDenialTracker returns a tracker with the default threshold of 3.
func NewDenialTracker() *DenialTracker {
	return &DenialTracker{
		counts:    make(map[string]int),
		threshold: defaultFallbackThreshold,
	}
}

// NewDenialTrackerWithThreshold returns a tracker with a custom threshold.
func NewDenialTrackerWithThreshold(threshold int) *DenialTracker {
	if threshold < 1 {
		threshold = defaultFallbackThreshold
	}
	return &DenialTracker{
		counts:    make(map[string]int),
		threshold: threshold,
	}
}

// RecordDenial increments the consecutive denial counter for toolName.
func (dt *DenialTracker) RecordDenial(toolName string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.counts[toolName]++
}

// ConsecutiveDenials returns how many times toolName was denied in a row.
func (dt *DenialTracker) ConsecutiveDenials(toolName string) int {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.counts[toolName]
}

// ShouldFallbackToPrompt returns true when the consecutive denial count
// meets or exceeds the threshold, signalling the caller to switch from
// automatic denial to interactive prompting.
func (dt *DenialTracker) ShouldFallbackToPrompt(toolName string) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.counts[toolName] >= dt.threshold
}

// Reset clears the denial counter for toolName (e.g. after a successful allow).
func (dt *DenialTracker) Reset(toolName string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	delete(dt.counts, toolName)
}
