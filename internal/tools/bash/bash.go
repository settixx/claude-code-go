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

// dangerousPatterns are command prefixes/fragments that are blocked outright.
var dangerousPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs.",
	"dd if=/dev/zero",
	"dd if=/dev/random",
	":(){:|:&};:",
	"> /dev/sda",
	"chmod -R 777 /",
}

// Tool executes shell commands.
type Tool struct {
	toolutil.BaseTool
}

// New creates a ready-to-use BashTool.
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
	return isDangerous(cmd)
}

// Call executes the shell command and returns combined stdout+stderr.
func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	command, err := toolutil.RequireString(input, "command")
	if err != nil {
		return nil, err
	}

	if isDangerous(command) {
		return nil, fmt.Errorf("command blocked for safety: %q", command)
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
		result := fmt.Sprintf("Exit code: %d\n%s", exitCode, output)
		return &types.ToolResult{Data: result}, nil
	}

	return &types.ToolResult{Data: output}, nil
}

func isDangerous(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	for _, pat := range dangerousPatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
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
