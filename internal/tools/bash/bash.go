package bash

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName         = "Bash"
	defaultTimeoutMs = 120_000
	maxResultChars   = 100_000
)

// Tool executes shell commands with security validation.
type Tool struct {
	toolutil.BaseTool
	readOnlyMode bool
}

// New creates a ready-to-use BashTool in normal mode.
func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"bash", "shell", "terminal"},
			ToolSearchHint:  "run shell commands, execute scripts",
			ToolMaxChars:    maxResultChars,
			ReadOnly:        false,
			Destructive:     false,
			ConcurrencySafe: false,
		},
	}
}

// NewReadOnly creates a BashTool that only allows read-only commands.
func NewReadOnly() *Tool {
	t := New()
	t.readOnlyMode = true
	t.ReadOnly = true
	return t
}

// Description returns a human-readable description for the model.
func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Execute a shell command on the local machine. " +
		"Use for running scripts, installing packages, compiling code, " +
		"or any operation that requires a shell.", nil
}

// InputSchema returns the JSON Schema for the tool's input.
func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in milliseconds (default 120000)",
			},
		},
		Required: []string{"command"},
	}
}

// IsDestructive inspects the command to decide whether it looks destructive.
func (t *Tool) IsDestructive(input map[string]interface{}) bool {
	cmd := toolutil.OptionalString(input, "command", "")
	result := ValidateCommand(cmd)
	return result.Behavior == SecurityDeny
}

// Call executes the shell command and returns combined stdout+stderr.
func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	command, err := toolutil.RequireString(input, "command")
	if err != nil {
		return nil, err
	}

	if reject := t.checkSecurity(command); reject != nil {
		return nil, reject
	}

	timeoutMs := toolutil.OptionalInt(input, "timeout", defaultTimeoutMs)
	timeout := time.Duration(timeoutMs) * time.Millisecond

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	output := buildOutput(stdout.String(), stderr.String())
	output = toolutil.TruncateResult(output, maxResultChars)

	if runErr != nil {
		exitCode := extractExitCode(runErr)
		output = annotateWithSemantics(command, exitCode, stdout.String(), stderr.String(), output)
		result := fmt.Sprintf("Exit code: %d\n%s", exitCode, output)
		return &types.ToolResult{Data: result}, nil
	}

	return &types.ToolResult{Data: output}, nil
}

// checkSecurity runs the full validator chain and, in read-only mode,
// the read-only allow list. Returns an error if the command is blocked.
func (t *Tool) checkSecurity(command string) error {
	sec := ValidateCommand(command)
	if sec.Behavior == SecurityDeny {
		return fmt.Errorf("command blocked for safety: %s", sec.Message)
	}

	if t.readOnlyMode {
		ro := ValidateReadOnly(command)
		if ro.Behavior != SecurityAllow {
			return fmt.Errorf("command blocked in read-only mode: %s", ro.Message)
		}
	}

	return nil
}

// annotateWithSemantics adds context for commands whose non-zero exit codes
// have special meaning (e.g., grep returns 1 for "no match").
func annotateWithSemantics(command string, exitCode int, stdout, stderr, output string) string {
	semantic := GetCommandSemantic(command)
	if semantic == nil {
		return output
	}
	isErr, msg := semantic(exitCode, stdout, stderr)
	if isErr {
		return output
	}
	return output + "\n[note: " + msg + "]"
}

func buildOutput(stdout, stderr string) string {
	var parts []string
	if stdout != "" {
		parts = append(parts, stdout)
	}
	if stderr != "" {
		parts = append(parts, "STDERR:\n"+stderr)
	}
	if len(parts) == 0 {
		return "(no output)"
	}
	return strings.Join(parts, "\n")
}

func extractExitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}
