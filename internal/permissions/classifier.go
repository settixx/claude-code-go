package permissions

import (
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// Risk levels returned by ClassifyRisk.
const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"
)

var readOnlyTools = map[string]bool{
	"FileRead":   true,
	"Glob":       true,
	"Grep":       true,
	"LS":         true,
	"Search":     true,
	"WebFetch":   true,
	"TaskOutput": true,
}

var safeBashPrefixes = []string{
	"ls", "cat", "head", "tail", "wc", "grep", "rg", "find",
	"echo", "pwd", "whoami", "env", "printenv", "which", "type",
	"git status", "git log", "git diff", "git show", "git branch",
	"git remote", "git tag",
}

var dangerousBashPrefixes = []string{
	"rm ", "rm\t", "rmdir ", "chmod ", "chown ", "chgrp ",
	"mv ", "sudo ", "mkfs", "dd ", "shutdown", "reboot",
	"git push", "git reset --hard", "git clean -f",
	"curl ", "wget ",
}

// Classifier uses heuristics to decide whether a tool invocation should be
// allowed, denied, or requires user confirmation.
type Classifier struct{}

// NewClassifier returns a new Classifier.
func NewClassifier() *Classifier {
	return &Classifier{}
}

// Classify returns a PermissionBehavior for the given tool invocation.
func (c *Classifier) Classify(toolName string, input map[string]interface{}) types.PermissionBehavior {
	if readOnlyTools[toolName] {
		return types.BehaviorAllow
	}

	if toolName == "FileWrite" || toolName == "FileEdit" || toolName == "NotebookEdit" {
		return c.classifyFileWrite(input)
	}

	if toolName == "Bash" || toolName == "PowerShell" {
		return c.classifyBash(input)
	}

	return types.BehaviorAsk
}

// ClassifyRisk returns a human-readable risk level for a tool invocation.
func (c *Classifier) ClassifyRisk(toolName string, input map[string]interface{}) string {
	if readOnlyTools[toolName] {
		return RiskLow
	}

	if toolName == "Bash" || toolName == "PowerShell" {
		return c.classifyBashRisk(input)
	}

	if toolName == "FileWrite" || toolName == "FileEdit" || toolName == "NotebookEdit" {
		return c.classifyFileWriteRisk(input)
	}

	return RiskMedium
}

// IsToolReadOnly reports whether the tool is inherently read-only.
func IsToolReadOnly(toolName string) bool {
	return readOnlyTools[toolName]
}

func (c *Classifier) classifyFileWrite(input map[string]interface{}) types.PermissionBehavior {
	path, _ := input["file_path"].(string)
	if path == "" {
		path, _ = input["path"].(string)
	}
	if path == "" {
		return types.BehaviorAsk
	}

	if strings.HasPrefix(path, "/etc/") || strings.HasPrefix(path, "/usr/") || strings.HasPrefix(path, "/sys/") {
		return types.BehaviorAsk
	}
	return types.BehaviorAllow
}

func (c *Classifier) classifyBash(input map[string]interface{}) types.PermissionBehavior {
	cmd := extractCommand(input)
	if cmd == "" {
		return types.BehaviorAsk
	}

	for _, prefix := range dangerousBashPrefixes {
		if strings.HasPrefix(cmd, prefix) {
			return types.BehaviorAsk
		}
	}

	for _, prefix := range safeBashPrefixes {
		if cmd == prefix || strings.HasPrefix(cmd, prefix+" ") || strings.HasPrefix(cmd, prefix+"\t") {
			return types.BehaviorAllow
		}
	}

	return types.BehaviorAsk
}

func (c *Classifier) classifyBashRisk(input map[string]interface{}) string {
	cmd := extractCommand(input)
	if cmd == "" {
		return RiskMedium
	}

	for _, prefix := range dangerousBashPrefixes {
		if strings.HasPrefix(cmd, prefix) {
			return RiskHigh
		}
	}

	for _, prefix := range safeBashPrefixes {
		if cmd == prefix || strings.HasPrefix(cmd, prefix+" ") || strings.HasPrefix(cmd, prefix+"\t") {
			return RiskLow
		}
	}

	return RiskMedium
}

func (c *Classifier) classifyFileWriteRisk(input map[string]interface{}) string {
	path, _ := input["file_path"].(string)
	if path == "" {
		path, _ = input["path"].(string)
	}

	if strings.HasPrefix(path, "/etc/") || strings.HasPrefix(path, "/usr/") || strings.HasPrefix(path, "/sys/") {
		return RiskHigh
	}
	return RiskMedium
}
