package permissions

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// PermissionChoice encodes the four possible user responses to a permission prompt.
type PermissionChoice int

const (
	ChoiceAllow       PermissionChoice = iota
	ChoiceDeny
	ChoiceAlwaysAllow
	ChoiceAlwaysDeny
)

func (c PermissionChoice) String() string {
	switch c {
	case ChoiceAllow:
		return "allow"
	case ChoiceDeny:
		return "deny"
	case ChoiceAlwaysAllow:
		return "always_allow"
	case ChoiceAlwaysDeny:
		return "always_deny"
	default:
		return "unknown"
	}
}

// PermissionRequest describes a tool invocation that requires user approval.
type PermissionRequest struct {
	ToolName    string
	Input       map[string]interface{}
	Description string
	IsReadOnly  bool
	Risk        string // "low", "medium", "high"
}

// PermissionPrompter is the interface for asking the user about a permission.
type PermissionPrompter interface {
	Prompt(ctx context.Context, req PermissionRequest) (PermissionChoice, error)
}

// ---------- StdinPrompter ----------

const defaultPromptTimeout = 60 * time.Second

// StdinPrompter reads interactive permission responses from stdin.
// It shows a formatted permission box and supports allow/deny/always options.
// Times out after 60 seconds with an auto-deny.
type StdinPrompter struct {
	In      io.Reader
	Out     io.Writer
	Timeout time.Duration
}

// NewStdinPrompter returns a prompter that reads from os.Stdin and writes to os.Stderr.
func NewStdinPrompter() *StdinPrompter {
	return &StdinPrompter{
		In:      os.Stdin,
		Out:     os.Stderr,
		Timeout: defaultPromptTimeout,
	}
}

// Prompt displays the permission request and waits for user input.
// Ctrl+C / EOF / timeout all resolve to ChoiceDeny.
func (p *StdinPrompter) Prompt(ctx context.Context, req PermissionRequest) (PermissionChoice, error) {
	formatted := FormatPermissionRequest(req)
	fmt.Fprintln(p.Out, formatted)

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultPromptTimeout
	}

	type readResult struct {
		text string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		scanner := bufio.NewScanner(p.In)
		if scanner.Scan() {
			ch <- readResult{text: scanner.Text()}
			return
		}
		ch <- readResult{err: scanner.Err()}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ChoiceDeny, ctx.Err()
	case <-timer.C:
		fmt.Fprintln(p.Out, "\nPrompt timed out — auto-denying.")
		return ChoiceDeny, nil
	case res := <-ch:
		if res.err != nil {
			return ChoiceDeny, fmt.Errorf("reading permission response: %w", res.err)
		}
		return parseChoice(res.text), nil
	}
}

func parseChoice(raw string) PermissionChoice {
	answer := strings.TrimSpace(raw)
	switch answer {
	case "a", "y", "yes":
		return ChoiceAllow
	case "d", "n", "no":
		return ChoiceDeny
	case "A":
		return ChoiceAlwaysAllow
	case "D":
		return ChoiceAlwaysDeny
	default:
		return ChoiceDeny
	}
}

// ---------- NonInteractivePrompter ----------

// NonInteractivePrompter auto-decides permissions for headless/CI environments.
// Read-only tools are allowed; everything else is denied.
type NonInteractivePrompter struct {
	Logger *slog.Logger
}

// NewNonInteractivePrompter creates a prompter suitable for CI/headless use.
func NewNonInteractivePrompter(logger *slog.Logger) *NonInteractivePrompter {
	if logger == nil {
		logger = slog.Default()
	}
	return &NonInteractivePrompter{Logger: logger}
}

// Prompt auto-allows read-only tools and denies everything else.
func (p *NonInteractivePrompter) Prompt(_ context.Context, req PermissionRequest) (PermissionChoice, error) {
	if req.IsReadOnly {
		p.Logger.Info("non-interactive: auto-allow read-only tool",
			"tool", req.ToolName)
		return ChoiceAllow, nil
	}
	p.Logger.Warn("non-interactive: auto-deny non-read-only tool",
		"tool", req.ToolName, "risk", req.Risk)
	return ChoiceDeny, nil
}
