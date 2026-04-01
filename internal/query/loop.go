package query

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/errors"
	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/types"
)

// loopConfig bundles the dependencies that runLoop needs.
type loopConfig struct {
	client          interfaces.LLMClient
	executor        interfaces.ToolExecutor
	renderer        interfaces.Renderer
	permissions     interfaces.PermissionChecker
	budget          *Budget
	query           types.QueryConfig
	maxTurns        int
	maxAutoContinue int
	hooks           []StopHook
}

const defaultMaxAutoContinue = 5

// runLoop is the core conversation loop. It sends messages to the LLM,
// collects the response, executes any tool_use blocks, appends tool_result
// messages, and repeats until a terminal condition is met.
//
// Returns the final assistant APIMessage and the stop reason.
func runLoop(ctx context.Context, cfg loopConfig, history *History) (*types.APIMessage, StopReason, error) {
	turn := 0
	autoContinueCount := 0
	maxAC := cfg.maxAutoContinue
	if maxAC <= 0 {
		maxAC = defaultMaxAutoContinue
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, StopReasonContextCancel, nil
		}

		if cfg.maxTurns > 0 && turn >= cfg.maxTurns {
			slog.Info("max turns reached", "turns", turn)
			return nil, StopReasonMaxTurns, nil
		}

		if cfg.budget != nil && cfg.budget.Exhausted() {
			return nil, StopReasonBudgetExhaust, nil
		}

		apiMessages := history.PrepareForAPI()

		response, err := streamAndAssemble(ctx, cfg, apiMessages)
		if err != nil {
			if ctx.Err() != nil {
				return nil, StopReasonContextCancel, nil
			}
			return nil, StopReasonAborted, err
		}

		if response.Usage != nil {
			cfg.budget.RecordUsage(*response.Usage)
		}

		assistantMsg := AssistantMessageFromAPI(response)
		history.Append(assistantMsg)
		cfg.renderer.RenderMessage(assistantMsg)

		decision := ShouldStop(ctx, response, cfg.budget)
		if decision.Stop {
			cont := RunStopHooks(ctx, cfg.hooks, response, func(m types.Message) {
				history.Append(m)
			})
			if cont {
				turn++
				continue
			}
			return response, decision.Reason, nil
		}

		if decision.AutoContinue {
			autoContinueCount++
			if autoContinueCount > maxAC {
				slog.Warn("max auto-continue reached", "count", autoContinueCount)
				return response, StopReasonMaxTokens, nil
			}
			slog.Info("auto-continuing after max_tokens", "count", autoContinueCount)
			history.Append(NewContinueMessage())
			turn++
			continue
		}

		toolResults, err := executeToolBlocks(ctx, cfg.executor, response, cfg.renderer, cfg.permissions)
		if err != nil {
			return response, StopReasonAborted, err
		}

		resultMsg := NewToolResultMessage(toolResults)
		history.Append(resultMsg)

		autoContinueCount = 0
		turn++
	}
}

// streamAndAssemble opens a streaming connection to the LLM and
// reassembles the incremental events into a complete APIMessage.
func streamAndAssemble(ctx context.Context, cfg loopConfig, messages []types.Message) (*types.APIMessage, error) {
	cfg.renderer.RenderSpinner("Thinking…")
	defer cfg.renderer.StopSpinner()

	ch, err := cfg.client.Stream(ctx, cfg.query, messages)
	if err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}

	return assembleStream(ctx, ch, cfg.renderer)
}

// assembleStream consumes stream events and builds the final APIMessage.
func assembleStream(ctx context.Context, ch <-chan types.StreamEvent, renderer interfaces.Renderer) (*types.APIMessage, error) {
	var result *types.APIMessage
	blocks := make(map[int]*types.ContentBlock)
	var lastUsage *types.Usage

	for {
		select {
		case <-ctx.Done():
			if result != nil {
				if partial, err := finaliseAssembly(result, blocks, lastUsage); err == nil && partial != nil {
					return partial, nil
				}
			}
			return result, ctx.Err()
		case evt, ok := <-ch:
			if !ok {
				return finaliseAssembly(result, blocks, lastUsage)
			}
			if err := processEvent(evt, &result, blocks, &lastUsage, renderer); err != nil {
				return result, err
			}
		}
	}
}

func processEvent(
	evt types.StreamEvent,
	result **types.APIMessage,
	blocks map[int]*types.ContentBlock,
	lastUsage **types.Usage,
	renderer interfaces.Renderer,
) error {
	switch evt.Type {
	case types.EventMessageStart:
		if evt.Message != nil {
			clone := *evt.Message
			clone.Content = nil
			*result = &clone
		}

	case types.EventContentBlockStart:
		if evt.ContentBlock != nil {
			b := *evt.ContentBlock
			blocks[evt.Index] = &b
		}

	case types.EventContentBlockDelta:
		applyDelta(blocks, evt.Index, evt.Delta, renderer)

	case types.EventContentBlockStop:
		// block is finalised — nothing extra to do

	case types.EventMessageDelta:
		if *result != nil && evt.Delta != nil {
			if evt.Delta.StopReason != "" {
				(*result).StopReason = evt.Delta.StopReason
			}
		}
		if evt.Usage != nil {
			*lastUsage = evt.Usage
		}

	case types.EventError:
		if evt.Error != nil {
			return &errors.APIError{
				Type:    evt.Error.Type,
				Message: evt.Error.Message,
			}
		}

	case types.EventPing, types.EventMessageStop:
		// no-op
	}
	return nil
}

func applyDelta(blocks map[int]*types.ContentBlock, index int, delta *types.DeltaBlock, renderer interfaces.Renderer) {
	if delta == nil {
		return
	}
	b, ok := blocks[index]
	if !ok {
		return
	}
	switch {
	case delta.Text != "":
		b.Text += delta.Text
		renderer.RenderMessage(types.Message{
			Type: types.MsgProgress,
			Text: delta.Text,
		})
	case delta.PartialJSON != "":
		b.Text += delta.PartialJSON
	case delta.Thinking != "":
		b.Thinking += delta.Thinking
	}
}

func finaliseAssembly(result *types.APIMessage, blocks map[int]*types.ContentBlock, usage *types.Usage) (*types.APIMessage, error) {
	if result == nil {
		return nil, fmt.Errorf("stream ended without message_start")
	}
	result.Content = orderedBlocks(blocks)
	if usage != nil {
		result.Usage = usage
	}
	return result, nil
}

func orderedBlocks(m map[int]*types.ContentBlock) []types.ContentBlock {
	if len(m) == 0 {
		return nil
	}
	maxIdx := 0
	for k := range m {
		if k > maxIdx {
			maxIdx = k
		}
	}
	out := make([]types.ContentBlock, 0, len(m))
	for i := 0; i <= maxIdx; i++ {
		if b, ok := m[i]; ok {
			out = append(out, *b)
		}
	}
	return out
}

// executeToolBlocks finds all tool_use content blocks in the response,
// executes them (concurrently where safe), and returns tool_result blocks.
func executeToolBlocks(
	ctx context.Context,
	executor interfaces.ToolExecutor,
	response *types.APIMessage,
	renderer interfaces.Renderer,
	permChecker interfaces.PermissionChecker,
) ([]types.ContentBlock, error) {
	calls := extractToolCalls(response)
	if len(calls) == 0 {
		return nil, nil
	}

	concurrent, sequential := partitionByConcurrency(calls, executor)

	var results []types.ContentBlock

	if len(concurrent) > 0 {
		r, err := executeConcurrent(ctx, executor, concurrent, renderer, permChecker)
		if err != nil {
			return results, err
		}
		results = append(results, r...)
	}

	for _, tc := range sequential {
		r, err := executeSingle(ctx, executor, tc, renderer, permChecker)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}

	return results, nil
}

type toolCall struct {
	ID    string
	Name  string
	Input map[string]interface{}
}

func extractToolCalls(msg *types.APIMessage) []toolCall {
	var out []toolCall
	for _, block := range msg.Content {
		if block.Type != types.ContentToolUse {
			continue
		}
		out = append(out, toolCall{
			ID:    block.ID,
			Name:  block.Name,
			Input: block.Input,
		})
	}
	return out
}

func partitionByConcurrency(calls []toolCall, executor interfaces.ToolExecutor) (concurrent, sequential []toolCall) {
	for _, tc := range calls {
		tool := executor.Find(tc.Name)
		if tool != nil && tool.IsConcurrencySafe(tc.Input) {
			concurrent = append(concurrent, tc)
		} else {
			sequential = append(sequential, tc)
		}
	}
	return
}

func executeConcurrent(
	ctx context.Context,
	executor interfaces.ToolExecutor,
	calls []toolCall,
	renderer interfaces.Renderer,
	permChecker interfaces.PermissionChecker,
) ([]types.ContentBlock, error) {
	type indexedResult struct {
		idx   int
		block types.ContentBlock
		err   error
	}

	results := make([]types.ContentBlock, len(calls))
	ch := make(chan indexedResult, len(calls))
	var wg sync.WaitGroup

	for i, tc := range calls {
		wg.Add(1)
		go func(idx int, tc toolCall) {
			defer wg.Done()
			block, err := executeSingle(ctx, executor, tc, renderer, permChecker)
			ch <- indexedResult{idx: idx, block: block, err: err}
		}(i, tc)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var firstErr error
	for ir := range ch {
		if ir.err != nil && firstErr == nil {
			firstErr = ir.err
		}
		results[ir.idx] = ir.block
	}

	return results, firstErr
}

func executeSingle(
	ctx context.Context,
	executor interfaces.ToolExecutor,
	tc toolCall,
	renderer interfaces.Renderer,
	permChecker interfaces.PermissionChecker,
) (types.ContentBlock, error) {
	slog.Debug("executing tool", "name", tc.Name, "id", tc.ID)

	if permChecker != nil {
		result := permChecker.Check(tc.Name, tc.Input)
		if !result.Allowed {
			return buildToolResultBlock(tc.ID, nil, &errors.PermissionError{
				ToolName: tc.Name,
				Reason:   result.Reason,
			}), nil
		}
	}

	renderer.RenderMessage(types.Message{
		Type: types.MsgProgress,
		Text: fmt.Sprintf("Running %s…", tc.Name),
	})

	result, err := executor.Execute(ctx, tc.Name, tc.Input)
	block := buildToolResultBlock(tc.ID, result, err)
	return block, nil
}

func buildToolResultBlock(toolUseID string, result *types.ToolResult, err error) types.ContentBlock {
	block := types.ContentBlock{
		Type:      types.ContentToolResult,
		ToolUseID: toolUseID,
	}

	if err != nil {
		block.IsError = true
		block.Content = []types.ContentBlock{{
			Type: types.ContentText,
			Text: err.Error(),
		}}
		return block
	}

	if result == nil {
		block.Content = []types.ContentBlock{{
			Type: types.ContentText,
			Text: "Tool returned no result.",
		}}
		return block
	}

	block.Content = []types.ContentBlock{{
		Type: types.ContentText,
		Text: formatToolData(result.Data),
	}}
	return block
}

func formatToolData(data interface{}) string {
	if data == nil {
		return ""
	}
	if s, ok := data.(string); ok {
		return s
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(raw)
}

// toolResultContent is a convenience to build a text result string from
// multiple content blocks.
func toolResultContent(blocks []types.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}
