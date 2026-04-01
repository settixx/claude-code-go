package types

import "testing"

func TestToAgentId(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOK  bool
		wantVal AgentId
	}{
		{
			name:    "valid simple id",
			input:   "a1234567890abcdef",
			wantOK:  true,
			wantVal: AgentId("a1234567890abcdef"),
		},
		{
			name:    "valid id with prefix segment",
			input:   "aprefix-1234567890abcdef",
			wantOK:  true,
			wantVal: AgentId("aprefix-1234567890abcdef"),
		},
		{
			name:    "valid id with multi-segment prefix",
			input:   "afoo-bar-1234567890abcdef",
			wantOK:  true,
			wantVal: AgentId("afoo-bar-1234567890abcdef"),
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "missing leading a",
			input:  "b1234567890abcdef",
			wantOK: false,
		},
		{
			name:   "hex too short",
			input:  "a1234567890abcde",
			wantOK: false,
		},
		{
			name:   "hex too long",
			input:  "a1234567890abcdef0",
			wantOK: false,
		},
		{
			name:   "non-hex characters in suffix",
			input:  "a1234567890abcdeg",
			wantOK: false,
		},
		{
			name:   "plain text",
			input:  "not-an-agent-id",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToAgentId(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ToAgentId(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && got != tt.wantVal {
				t.Errorf("ToAgentId(%q) = %q, want %q", tt.input, got, tt.wantVal)
			}
		})
	}
}
