package bash

import "strings"

// CommandSemantic interprets the exit code, stdout, and stderr of a command
// and returns whether it should be treated as an error along with a
// human-readable explanation. Commands like grep and diff use non-zero codes
// for non-error conditions (e.g., "no match" or "files differ").
type CommandSemantic func(exitCode int, stdout, stderr string) (isError bool, message string)

// GetCommandSemantic returns a semantic interpreter for the given command,
// or nil if the command uses standard exit code conventions (0 = success,
// non-zero = error).
func GetCommandSemantic(command string) CommandSemantic {
	base := ExtractBaseCommand(command)
	if fn, ok := semanticRegistry[base]; ok {
		return fn
	}
	return nil
}

var semanticRegistry = map[string]CommandSemantic{
	"grep":  grepSemantic,
	"egrep": grepSemantic,
	"fgrep": grepSemantic,
	"rg":    grepSemantic,
	"find":  findSemantic,
	"diff":  diffSemantic,
	"test":  testSemantic,
	"[":     testSemantic,
}

func grepSemantic(exitCode int, _, stderr string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, "match found"
	case exitCode == 1:
		return false, "no match — this is normal for grep, not an error"
	default:
		return true, "grep error (exit " + itoa(exitCode) + "): " + firstLine(stderr)
	}
}

func findSemantic(exitCode int, _, stderr string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, "find completed successfully"
	case exitCode == 1:
		return false, "find had partial failures — some paths may be inaccessible"
	default:
		return true, "find error (exit " + itoa(exitCode) + "): " + firstLine(stderr)
	}
}

func diffSemantic(exitCode int, _, stderr string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, "files are identical"
	case exitCode == 1:
		return false, "files differ — this is normal for diff, not an error"
	default:
		return true, "diff error (exit " + itoa(exitCode) + "): " + firstLine(stderr)
	}
}

func testSemantic(exitCode int, _, stderr string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, "condition is true"
	case exitCode == 1:
		return false, "condition is false — this is normal for test/[, not an error"
	default:
		return true, "test error (exit " + itoa(exitCode) + "): " + firstLine(stderr)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
