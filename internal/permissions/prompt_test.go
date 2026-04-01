package permissions

import "testing"

// ---------------------------------------------------------------------------
// PermissionChoice.String
// ---------------------------------------------------------------------------

func TestPermissionChoice_String(t *testing.T) {
	tests := []struct {
		choice PermissionChoice
		want   string
	}{
		{ChoiceAllow, "allow"},
		{ChoiceDeny, "deny"},
		{ChoiceAlwaysAllow, "always_allow"},
		{ChoiceAlwaysDeny, "always_deny"},
		{PermissionChoice(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.choice.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseChoice
// ---------------------------------------------------------------------------

func TestParseChoice(t *testing.T) {
	tests := []struct {
		input string
		want  PermissionChoice
	}{
		{"a", ChoiceAllow},
		{"y", ChoiceAllow},
		{"yes", ChoiceAllow},
		{"d", ChoiceDeny},
		{"n", ChoiceDeny},
		{"no", ChoiceDeny},
		{"A", ChoiceAlwaysAllow},
		{"D", ChoiceAlwaysDeny},
		{"", ChoiceDeny},
		{"random", ChoiceDeny},
		{"  y  ", ChoiceAllow},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseChoice(tt.input)
			if got != tt.want {
				t.Errorf("parseChoice(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
