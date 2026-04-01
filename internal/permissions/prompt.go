package permissions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// PromptResult encodes the three possible user responses.
type PromptResult int

const (
	// PromptAllow means the user approved this single invocation.
	PromptAllow PromptResult = iota
	// PromptDeny means the user rejected this invocation.
	PromptDeny
	// PromptAlways means the user wants an allow rule persisted.
	PromptAlways
)

// PrompterFunc is the signature used by PromptForPermission so callers can
// substitute a custom reader in tests or non-TTY environments.
type PrompterFunc func(toolName string, input map[string]interface{}) (PromptResult, error)

// PromptForPermission prints a summary of the tool invocation to stdout, then
// reads a y/n/always answer from stdin. Returns (true, nil) for allow,
// (false, nil) for deny. When the user answers "always", true is returned
// and the caller is expected to persist an allow rule.
func PromptForPermission(toolName string, input map[string]interface{}) (PromptResult, error) {
	return promptFromReader(os.Stdin, os.Stderr, toolName, input)
}

func promptFromReader(r io.Reader, w io.Writer, toolName string, input map[string]interface{}) (PromptResult, error) {
	fmt.Fprintf(w, "\n── Permission Request ──\n")
	fmt.Fprintf(w, "Tool:    %s\n", toolName)
	printInputSummary(w, input)
	fmt.Fprintf(w, "Allow this action? [y/n/always] ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return PromptDeny, fmt.Errorf("reading permission response: %w", err)
		}
		return PromptDeny, nil
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	switch answer {
	case "y", "yes":
		return PromptAllow, nil
	case "always", "a":
		return PromptAlways, nil
	default:
		return PromptDeny, nil
	}
}

func printInputSummary(w io.Writer, input map[string]interface{}) {
	interesting := []string{"command", "cmd", "file_path", "path", "query", "pattern"}
	for _, key := range interesting {
		v, ok := input[key]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		display := s
		if len(display) > 120 {
			display = display[:117] + "..."
		}
		fmt.Fprintf(w, "  %-10s %s\n", key+":", display)
	}
}
