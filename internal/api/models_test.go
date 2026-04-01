package api

import "testing"

func TestResolveModel(t *testing.T) {
	tests := []struct {
		alias string
		want  string
	}{
		{"opus", ModelClaude4Opus},
		{"sonnet", ModelClaude4Sonnet},
		{"haiku", ModelClaude35Haiku},
		{"claude-4-opus", ModelClaude4Opus},
		{"claude-4-sonnet", ModelClaude4Sonnet},
		{"claude-3.5-haiku", ModelClaude35Haiku},
		{"claude-3.5-sonnet", ModelClaude35Sonnet},
		{"claude-3-opus", ModelClaude3Opus},
		{"claude-3-sonnet", ModelClaude3Sonnet},
		{"claude-3-haiku", ModelClaude3Haiku},
		{"  Opus  ", ModelClaude4Opus},
		{"HAIKU", ModelClaude35Haiku},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got := ResolveModel(tt.alias)
			if got != tt.want {
				t.Errorf("ResolveModel(%q) = %q, want %q", tt.alias, got, tt.want)
			}
		})
	}
}

func TestResolveModelUnknown(t *testing.T) {
	unknown := "some-custom-model-v1"
	got := ResolveModel(unknown)
	if got != unknown {
		t.Errorf("ResolveModel(%q) = %q, want passthrough", unknown, got)
	}
}

func TestGetModelCapabilities(t *testing.T) {
	t.Run("known model by alias", func(t *testing.T) {
		caps := GetModelCapabilities("opus")
		if !caps.SupportsThinking {
			t.Error("opus should support thinking")
		}
		if !caps.SupportsTools {
			t.Error("opus should support tools")
		}
		if caps.MaxOutputTokens != 32000 {
			t.Errorf("opus MaxOutputTokens = %d, want 32000", caps.MaxOutputTokens)
		}
		if caps.ContextWindow != 200000 {
			t.Errorf("opus ContextWindow = %d, want 200000", caps.ContextWindow)
		}
	})

	t.Run("known model by full ID", func(t *testing.T) {
		caps := GetModelCapabilities(ModelClaude35Haiku)
		if caps.SupportsThinking {
			t.Error("haiku should not support thinking")
		}
		if caps.MaxOutputTokens != 8192 {
			t.Errorf("haiku MaxOutputTokens = %d, want 8192", caps.MaxOutputTokens)
		}
	})

	t.Run("unknown model returns defaults", func(t *testing.T) {
		caps := GetModelCapabilities("unknown-model-xyz")
		if caps.SupportsThinking {
			t.Error("unknown model should default to no thinking")
		}
		if !caps.SupportsTools {
			t.Error("unknown model should default to tools support")
		}
		if caps.MaxOutputTokens != 4096 {
			t.Errorf("unknown model MaxOutputTokens = %d, want 4096", caps.MaxOutputTokens)
		}
	})
}

func TestIsKnownModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"opus", true},
		{ModelClaude4Sonnet, true},
		{"nonexistent-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsKnownModel(tt.model); got != tt.want {
				t.Errorf("IsKnownModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
