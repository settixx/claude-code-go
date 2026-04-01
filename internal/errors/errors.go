package errors

import "fmt"

// AbortError signals intentional cancellation (Ctrl+C, timeout, etc.).
type AbortError struct {
	Message string
}

func (e *AbortError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "operation aborted"
}

func IsAbortError(err error) bool {
	_, ok := err.(*AbortError)
	return ok
}

// ToolError wraps an error that occurred during tool execution.
type ToolError struct {
	ToolName string
	Cause    error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %q: %v", e.ToolName, e.Cause)
}

func (e *ToolError) Unwrap() error { return e.Cause }

// PermissionError indicates a tool call was denied by the permission system.
type PermissionError struct {
	ToolName string
	Reason   string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied for %q: %s", e.ToolName, e.Reason)
}

// APIError wraps an error returned by the LLM API.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	Retryable  bool
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Type, e.Message)
}

// ValidationError indicates invalid tool input.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ConfigParseError indicates a malformed configuration file.
type ConfigParseError struct {
	FilePath      string
	Cause         error
	DefaultConfig interface{}
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse config %q: %v", e.FilePath, e.Cause)
}

func (e *ConfigParseError) Unwrap() error { return e.Cause }

// ShellError wraps a failed shell command.
type ShellError struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	Interrupted bool
}

func (e *ShellError) Error() string {
	return fmt.Sprintf("shell command failed (exit %d): %s", e.ExitCode, e.Stderr)
}

// ToError normalizes an unknown value into an error.
func ToError(v interface{}) error {
	if v == nil {
		return nil
	}
	if err, ok := v.(error); ok {
		return err
	}
	return fmt.Errorf("%v", v)
}

// ErrorMessage extracts a message string from an error-like value.
func ErrorMessage(v interface{}) string {
	if v == nil {
		return ""
	}
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("%v", v)
}
