package query

import (
	"context"
	"log/slog"

	"github.com/settixx/claude-code-go/internal/types"
)

// StopHook is called after the assistant produces a terminal response.
// Returning true means the conversation loop should continue with another
// iteration (e.g. the hook injected a follow-up user message). Returning
// false means the loop should terminate normally.
type StopHook interface {
	// Name returns a human-readable identifier for logging.
	Name() string

	// ShouldRun reports whether this hook applies to the given response.
	ShouldRun(response *types.APIMessage) bool

	// Run executes the hook logic. It may mutate the message history via the
	// provided append callback. It returns true if the loop must continue.
	Run(ctx context.Context, response *types.APIMessage, appendMsg func(types.Message)) (continueLoop bool, err error)
}

// StopReason describes why the conversation loop terminated.
type StopReason string

const (
	StopReasonEndTurn       StopReason = "end_turn"
	StopReasonMaxTokens     StopReason = "max_tokens"
	StopReasonBudgetExhaust StopReason = "budget_exhausted"
	StopReasonContextCancel StopReason = "context_cancelled"
	StopReasonMaxTurns      StopReason = "max_turns"
	StopReasonAborted       StopReason = "aborted"
)

// StopDecision bundles the result of ShouldStop.
type StopDecision struct {
	Stop         bool
	AutoContinue bool
	Reason       StopReason
}

// ShouldStop inspects the API response and surrounding context to decide
// whether the conversation loop should terminate.
func ShouldStop(ctx context.Context, response *types.APIMessage, budget *Budget) StopDecision {
	if ctx.Err() != nil {
		return StopDecision{Stop: true, Reason: StopReasonContextCancel}
	}

	if budget != nil && budget.Exhausted() {
		return StopDecision{Stop: true, Reason: StopReasonBudgetExhaust}
	}

	if response == nil {
		return StopDecision{Stop: true, Reason: StopReasonAborted}
	}

	if hasToolUse(response) {
		return StopDecision{Stop: false}
	}

	switch response.StopReason {
	case types.StopEndTurn:
		return StopDecision{Stop: true, Reason: StopReasonEndTurn}
	case types.StopMaxTokens:
		return StopDecision{Stop: false, AutoContinue: true, Reason: StopReasonMaxTokens}
	default:
		return StopDecision{Stop: true, Reason: StopReasonEndTurn}
	}
}

// RunStopHooks executes all matching stop hooks in order. If any hook
// requests continuation, the function returns true.
func RunStopHooks(ctx context.Context, hooks []StopHook, response *types.APIMessage, appendMsg func(types.Message)) bool {
	shouldContinue := false
	for _, h := range hooks {
		if !h.ShouldRun(response) {
			continue
		}
		slog.Debug("running stop hook", "hook", h.Name())

		cont, err := h.Run(ctx, response, appendMsg)
		if err != nil {
			slog.Error("stop hook failed", "hook", h.Name(), "error", err)
			continue
		}
		if cont {
			shouldContinue = true
		}
	}
	return shouldContinue
}

// hasToolUse reports whether the response contains at least one tool_use block.
func hasToolUse(msg *types.APIMessage) bool {
	for _, block := range msg.Content {
		if block.Type == types.ContentToolUse {
			return true
		}
	}
	return false
}
