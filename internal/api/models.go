package api

import "strings"

// Model ID constants for the Anthropic Claude family.
const (
	ModelClaude4Opus       = "claude-4-opus-20260301"
	ModelClaude4Sonnet     = "claude-4-sonnet-20260301"
	ModelClaude35Haiku     = "claude-3-5-haiku-20241022"
	ModelClaude35Sonnet    = "claude-3-5-sonnet-20241022"
	ModelClaude3Opus       = "claude-3-opus-20240229"
	ModelClaude3Sonnet     = "claude-3-sonnet-20240229"
	ModelClaude3Haiku      = "claude-3-haiku-20240307"
)

// modelAliases maps short convenience names to full model IDs.
var modelAliases = map[string]string{
	"opus":          ModelClaude4Opus,
	"sonnet":        ModelClaude4Sonnet,
	"haiku":         ModelClaude35Haiku,
	"claude-4-opus": ModelClaude4Opus,
	"claude-4-sonnet": ModelClaude4Sonnet,
	"claude-3.5-haiku": ModelClaude35Haiku,
	"claude-3.5-sonnet": ModelClaude35Sonnet,
	"claude-3-opus":   ModelClaude3Opus,
	"claude-3-sonnet": ModelClaude3Sonnet,
	"claude-3-haiku":  ModelClaude3Haiku,
}

// ModelCapabilities describes what features a given model supports.
type ModelCapabilities struct {
	SupportsThinking bool
	SupportsTools    bool
	MaxOutputTokens  int
	ContextWindow    int
}

// knownModels stores capability info keyed by the canonical model ID.
var knownModels = map[string]ModelCapabilities{
	ModelClaude4Opus: {
		SupportsThinking: true,
		SupportsTools:    true,
		MaxOutputTokens:  32000,
		ContextWindow:    200000,
	},
	ModelClaude4Sonnet: {
		SupportsThinking: true,
		SupportsTools:    true,
		MaxOutputTokens:  16000,
		ContextWindow:    200000,
	},
	ModelClaude35Haiku: {
		SupportsThinking: false,
		SupportsTools:    true,
		MaxOutputTokens:  8192,
		ContextWindow:    200000,
	},
	ModelClaude35Sonnet: {
		SupportsThinking: true,
		SupportsTools:    true,
		MaxOutputTokens:  8192,
		ContextWindow:    200000,
	},
	ModelClaude3Opus: {
		SupportsThinking: false,
		SupportsTools:    true,
		MaxOutputTokens:  4096,
		ContextWindow:    200000,
	},
	ModelClaude3Sonnet: {
		SupportsThinking: false,
		SupportsTools:    true,
		MaxOutputTokens:  4096,
		ContextWindow:    200000,
	},
	ModelClaude3Haiku: {
		SupportsThinking: false,
		SupportsTools:    true,
		MaxOutputTokens:  4096,
		ContextWindow:    200000,
	},
}

// ResolveModel maps a short alias or full model ID to the canonical model ID.
// If the input is already a known full ID, it is returned as-is.
func ResolveModel(alias string) string {
	lower := strings.ToLower(strings.TrimSpace(alias))
	if resolved, ok := modelAliases[lower]; ok {
		return resolved
	}
	return alias
}

// GetModelCapabilities returns the capabilities for a model, resolving aliases first.
// Returns a zero-value ModelCapabilities with sensible defaults for unknown models.
func GetModelCapabilities(model string) ModelCapabilities {
	resolved := ResolveModel(model)
	if caps, ok := knownModels[resolved]; ok {
		return caps
	}
	return ModelCapabilities{
		SupportsThinking: false,
		SupportsTools:    true,
		MaxOutputTokens:  4096,
		ContextWindow:    200000,
	}
}

// IsKnownModel returns true if the model (or its alias) is in the known set.
func IsKnownModel(model string) bool {
	resolved := ResolveModel(model)
	_, ok := knownModels[resolved]
	return ok
}
