package permissions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// writeTools enumerates tool names that perform write/mutate operations.
var writeTools = map[string]bool{
	"FileWrite":    true,
	"FileEdit":     true,
	"NotebookEdit": true,
	"Bash":         true,
	"PowerShell":   true,
	"TaskStop":     true,
	"WebFetch":     true,
}

// editOnlyTools are write tools that only modify files (no shell execution).
var editOnlyTools = map[string]bool{
	"FileWrite":    true,
	"FileEdit":     true,
	"NotebookEdit": true,
}

// DecisionEntry records a single permission decision for audit / status display.
type DecisionEntry struct {
	Timestamp time.Time
	ToolName  string
	Decision  PermissionChoice
	Reason    string
}

// Checker implements permission checking. It is safe for concurrent use.
type Checker struct {
	mu         sync.RWMutex
	mode       types.PermissionMode
	rules      *RuleSet
	classifier *Classifier
	denials    *DenialTracker
	prompter   PermissionPrompter

	logMu   sync.Mutex
	history []DecisionEntry
}

// NewChecker creates a Checker with the given initial mode, rule set, and prompter.
// Nil arguments get safe defaults (empty RuleSet, StdinPrompter).
func NewChecker(mode types.PermissionMode, rules *RuleSet) *Checker {
	if rules == nil {
		rules = NewRuleSet()
	}
	return &Checker{
		mode:       mode,
		rules:      rules,
		classifier: NewClassifier(),
		denials:    NewDenialTracker(),
		prompter:   NewStdinPrompter(),
	}
}

// SetPrompter replaces the active prompter (useful for switching to CI mode).
func (c *Checker) SetPrompter(p PermissionPrompter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prompter = p
}

// Check evaluates whether a tool invocation is permitted under the current
// mode and rule set. It never prompts — it only returns the static decision
// or signals that user confirmation is required.
func (c *Checker) Check(toolName string, input map[string]interface{}) types.PermissionResult {
	c.mu.RLock()
	mode := c.mode
	c.mu.RUnlock()

	switch mode {
	case types.PermBypassPermissions, types.PermDontAsk:
		return allowed("mode " + string(mode) + " permits all actions")
	case types.PermPlan:
		return c.checkPlanMode(toolName)
	case types.PermAcceptEdits:
		return c.checkAcceptEditsMode(toolName, input)
	case types.PermAuto:
		return c.checkAutoMode(toolName, input)
	default:
		return c.checkDefaultMode(toolName, input)
	}
}

// CheckWithPrompt performs the full check-then-prompt flow. When the static
// check returns "ask", it builds a PermissionRequest, prompts the user, and
// applies "Always Allow" / "Always Deny" to the session rule set.
func (c *Checker) CheckWithPrompt(ctx context.Context, toolName string, input map[string]interface{}) (types.PermissionResult, error) {
	result := c.Check(toolName, input)
	if result.Allowed {
		c.recordDecision(toolName, ChoiceAllow, result.Reason)
		return result, nil
	}
	if result.Reason != "" && !isAskReason(result.Reason) {
		c.recordDecision(toolName, ChoiceDeny, result.Reason)
		return result, nil
	}

	req := c.buildRequest(toolName, input)

	c.mu.RLock()
	prompter := c.prompter
	c.mu.RUnlock()

	choice, err := prompter.Prompt(ctx, req)
	if err != nil {
		c.recordDecision(toolName, ChoiceDeny, "prompt error: "+err.Error())
		return denied("prompt error: " + err.Error()), err
	}

	return c.applyChoice(toolName, input, choice), nil
}

// BuildRequest creates a PermissionRequest for display / external consumers.
func (c *Checker) BuildRequest(toolName string, input map[string]interface{}) PermissionRequest {
	return c.buildRequest(toolName, input)
}

// DecisionHistory returns a copy of all recorded decisions in this session.
func (c *Checker) DecisionHistory() []DecisionEntry {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	out := make([]DecisionEntry, len(c.history))
	copy(out, c.history)
	return out
}

// Mode returns the current permission mode.
func (c *Checker) Mode() types.PermissionMode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mode
}

// SetMode changes the active permission mode.
func (c *Checker) SetMode(mode types.PermissionMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mode = mode
}

// Rules returns the underlying RuleSet.
func (c *Checker) Rules() *RuleSet {
	return c.rules
}

// DenialTracker exposes the denial tracker.
func (c *Checker) DenialTracker() *DenialTracker {
	return c.denials
}

// ---------- internal ----------

func (c *Checker) buildRequest(toolName string, input map[string]interface{}) PermissionRequest {
	risk := c.classifier.ClassifyRisk(toolName, input)
	readOnly := IsToolReadOnly(toolName)

	desc := buildDescription(toolName, input)

	return PermissionRequest{
		ToolName:    toolName,
		Input:       input,
		Description: desc,
		IsReadOnly:  readOnly,
		Risk:        risk,
	}
}

func (c *Checker) applyChoice(toolName string, input map[string]interface{}, choice PermissionChoice) types.PermissionResult {
	switch choice {
	case ChoiceAllow:
		c.denials.Reset(toolName)
		c.recordDecision(toolName, ChoiceAllow, "user allowed")
		return allowed("user allowed")

	case ChoiceAlwaysAllow:
		c.denials.Reset(toolName)
		cmd := extractCommand(input)
		c.rules.AddAlwaysAllow(toolName, cmd)
		c.recordDecision(toolName, ChoiceAlwaysAllow, "user set always-allow")
		return allowed("user set always-allow for session")

	case ChoiceAlwaysDeny:
		c.denials.RecordDenial(toolName)
		cmd := extractCommand(input)
		c.rules.AddAlwaysDeny(toolName, cmd)
		c.recordDecision(toolName, ChoiceAlwaysDeny, "user set always-deny")
		return denied("user set always-deny for session")

	default:
		c.denials.RecordDenial(toolName)
		c.recordDecision(toolName, ChoiceDeny, "user denied")
		return denied("user denied")
	}
}

func (c *Checker) recordDecision(toolName string, choice PermissionChoice, reason string) {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	c.history = append(c.history, DecisionEntry{
		Timestamp: time.Now(),
		ToolName:  toolName,
		Decision:  choice,
		Reason:    reason,
	})
}

func isAskReason(reason string) bool {
	return reason == "requires user confirmation" ||
		reason == "no matching rule; requires user confirmation"
}

func buildDescription(toolName string, input map[string]interface{}) string {
	cmd := extractCommand(input)
	if cmd != "" {
		return fmt.Sprintf("Execute %s: %s", toolName, cmd)
	}

	path, _ := input["file_path"].(string)
	if path == "" {
		path, _ = input["path"].(string)
	}
	if path != "" {
		return fmt.Sprintf("%s on %s", toolName, path)
	}

	return fmt.Sprintf("Run tool %s", toolName)
}

func (c *Checker) checkPlanMode(toolName string) types.PermissionResult {
	if writeTools[toolName] {
		return denied("plan mode does not permit write tool " + toolName)
	}
	return allowed("plan mode permits read-only tool " + toolName)
}

func (c *Checker) checkAcceptEditsMode(toolName string, input map[string]interface{}) types.PermissionResult {
	if !writeTools[toolName] {
		return allowed("acceptEdits mode permits read-only tool " + toolName)
	}
	if editOnlyTools[toolName] {
		return allowed("acceptEdits mode permits file-editing tool " + toolName)
	}
	if toolName == "Bash" || toolName == "PowerShell" {
		behavior := c.classifier.classifyBash(input)
		if behavior == types.BehaviorAllow {
			return allowed("acceptEdits mode permits safe bash command")
		}
		return denied("acceptEdits mode denies potentially dangerous bash command")
	}
	return denied("acceptEdits mode denies non-edit write tool " + toolName)
}

func (c *Checker) checkAutoMode(toolName string, input map[string]interface{}) types.PermissionResult {
	behavior := c.rules.Evaluate(toolName, input)
	if behavior == types.BehaviorDeny {
		return denied("denied by rule")
	}
	if behavior == types.BehaviorAllow {
		c.denials.Reset(toolName)
		return allowed("allowed by rule")
	}

	behavior = c.classifier.Classify(toolName, input)
	switch behavior {
	case types.BehaviorAllow:
		c.denials.Reset(toolName)
		return allowed("auto-classified as safe")
	case types.BehaviorDeny:
		c.denials.RecordDenial(toolName)
		return denied("auto-classified as dangerous")
	default:
		return types.PermissionResult{Allowed: false, Reason: "requires user confirmation"}
	}
}

func (c *Checker) checkDefaultMode(toolName string, input map[string]interface{}) types.PermissionResult {
	behavior := c.rules.Evaluate(toolName, input)
	switch behavior {
	case types.BehaviorAllow:
		c.denials.Reset(toolName)
		return allowed("allowed by rule")
	case types.BehaviorDeny:
		c.denials.RecordDenial(toolName)
		return denied("denied by rule")
	default:
		return types.PermissionResult{Allowed: false, Reason: "no matching rule; requires user confirmation"}
	}
}

func allowed(reason string) types.PermissionResult {
	return types.PermissionResult{Allowed: true, Reason: reason}
}

func denied(reason string) types.PermissionResult {
	return types.PermissionResult{Allowed: false, Reason: reason}
}
