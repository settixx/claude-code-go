package errors

import (
	"fmt"
	"testing"
)

func TestAbortError(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		err := &AbortError{Message: "user cancelled"}
		if err.Error() != "user cancelled" {
			t.Errorf("Error() = %q, want %q", err.Error(), "user cancelled")
		}
	})

	t.Run("empty message uses default", func(t *testing.T) {
		err := &AbortError{}
		if err.Error() != "operation aborted" {
			t.Errorf("Error() = %q, want %q", err.Error(), "operation aborted")
		}
	})
}

func TestToolError(t *testing.T) {
	cause := fmt.Errorf("file not found")
	err := &ToolError{ToolName: "FileRead", Cause: cause}

	want := `tool "FileRead": file not found`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap() did not return the original cause")
	}
}

func TestPermissionError(t *testing.T) {
	err := &PermissionError{ToolName: "Bash", Reason: "dangerous command"}
	want := `permission denied for "Bash": dangerous command`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 429, Type: "rate_limit_error", Message: "too many requests"}
	want := "API error 429 (rate_limit_error): too many requests"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestValidationError(t *testing.T) {
	t.Run("with field", func(t *testing.T) {
		err := &ValidationError{Field: "command", Message: "required"}
		want := `validation error on "command": required`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := &ValidationError{Message: "bad input"}
		want := "validation error: bad input"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})
}

func TestConfigParseError(t *testing.T) {
	cause := fmt.Errorf("invalid JSON")
	err := &ConfigParseError{FilePath: "/etc/config.json", Cause: cause}

	want := `failed to parse config "/etc/config.json": invalid JSON`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	if err.Unwrap() != cause {
		t.Error("Unwrap() did not return the original cause")
	}
}

func TestShellError(t *testing.T) {
	err := &ShellError{Stdout: "out", Stderr: "command not found", ExitCode: 127}
	want := "shell command failed (exit 127): command not found"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestIsAbortError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"abort error", &AbortError{Message: "abort"}, true},
		{"tool error", &ToolError{ToolName: "Bash"}, false},
		{"plain error", fmt.Errorf("something"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAbortError(tt.err); got != tt.want {
				t.Errorf("IsAbortError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToError(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if err := ToError(nil); err != nil {
			t.Errorf("ToError(nil) = %v, want nil", err)
		}
	})

	t.Run("error passthrough", func(t *testing.T) {
		original := fmt.Errorf("original")
		if err := ToError(original); err != original {
			t.Errorf("ToError(error) returned different error")
		}
	})

	t.Run("string converted to error", func(t *testing.T) {
		err := ToError("boom")
		if err == nil {
			t.Fatal("ToError(string) = nil, want error")
		}
		if err.Error() != "boom" {
			t.Errorf("Error() = %q, want %q", err.Error(), "boom")
		}
	})

	t.Run("int converted to error", func(t *testing.T) {
		err := ToError(42)
		if err == nil {
			t.Fatal("ToError(int) = nil, want error")
		}
		if err.Error() != "42" {
			t.Errorf("Error() = %q, want %q", err.Error(), "42")
		}
	})
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, ""},
		{"error value", fmt.Errorf("oops"), "oops"},
		{"string value", "hello", "hello"},
		{"int value", 99, "99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorMessage(tt.input)
			if got != tt.want {
				t.Errorf("ErrorMessage(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
