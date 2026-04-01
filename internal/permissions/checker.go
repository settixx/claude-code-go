package permissions

import (
	"sync"

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

// Checker implements interfaces.PermissionChecker. It is safe for concurrent
// use; mode and rules are guarded by an RWMutex.
type Checker struct {
	mu         sync.RWMutex
	mode       types.PermissionMode
	rules      *RuleSet
	classifier *Classifier
	denials    *DenialTracker
}

// NewChecker creates a Checker with the given initial mode and rule set.
// If rules is nil, an empty RuleSet is used.
func NewChecker(mode types.PermissionMode, rules *RuleSet) *Checker {
	if rules == nil {
		rules = NewRuleSet()
	}
	return &Checker{
		mode:       mode,
		rules:      rules,
		classifier: NewClassifier(),
		denials:    NewDenialTracker(),
	}
}

// Check evaluates whether a tool invocation is permitted under the current
// mode and rule set. The returned PermissionResult always has a human-readable
// Reason.
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

// Rules returns the underlying RuleSet so callers can add runtime rules
// (e.g. when the user answers "always" to a prompt).
func (c *Checker) Rules() *RuleSet {
	return c.rules
}

// DenialTracker exposes the denial tracker for callers that need to record
// or query denial history.
func (c *Checker) DenialTracker() *DenialTracker {
	return c.denials
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
