package permissions

import "testing"

// ---------------------------------------------------------------------------
// DenialTracker
// ---------------------------------------------------------------------------

func TestDenialTracker_RecordAndCount(t *testing.T) {
	dt := NewDenialTracker()

	if dt.ConsecutiveDenials("Bash") != 0 {
		t.Error("initial count should be 0")
	}

	dt.RecordDenial("Bash")
	dt.RecordDenial("Bash")

	if dt.ConsecutiveDenials("Bash") != 2 {
		t.Errorf("count = %d, want 2", dt.ConsecutiveDenials("Bash"))
	}
}

func TestDenialTracker_ShouldFallbackToPrompt(t *testing.T) {
	dt := NewDenialTrackerWithThreshold(2)

	dt.RecordDenial("Bash")
	if dt.ShouldFallbackToPrompt("Bash") {
		t.Error("should not fallback after 1 denial (threshold=2)")
	}

	dt.RecordDenial("Bash")
	if !dt.ShouldFallbackToPrompt("Bash") {
		t.Error("should fallback after 2 denials (threshold=2)")
	}
}

func TestDenialTracker_Reset(t *testing.T) {
	dt := NewDenialTracker()
	dt.RecordDenial("Bash")
	dt.RecordDenial("Bash")
	dt.RecordDenial("Bash")

	dt.Reset("Bash")
	if dt.ConsecutiveDenials("Bash") != 0 {
		t.Error("count should be 0 after Reset")
	}
}

func TestDenialTracker_IndependentTools(t *testing.T) {
	dt := NewDenialTracker()
	dt.RecordDenial("Bash")
	dt.RecordDenial("FileWrite")
	dt.RecordDenial("FileWrite")

	if dt.ConsecutiveDenials("Bash") != 1 {
		t.Errorf("Bash count = %d, want 1", dt.ConsecutiveDenials("Bash"))
	}
	if dt.ConsecutiveDenials("FileWrite") != 2 {
		t.Errorf("FileWrite count = %d, want 2", dt.ConsecutiveDenials("FileWrite"))
	}
}

func TestDenialTracker_InvalidThreshold(t *testing.T) {
	dt := NewDenialTrackerWithThreshold(0)
	for i := 0; i < defaultFallbackThreshold; i++ {
		dt.RecordDenial("X")
	}
	if !dt.ShouldFallbackToPrompt("X") {
		t.Error("invalid threshold should default to 3")
	}
}
