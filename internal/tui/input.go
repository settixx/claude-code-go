package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// InputReader reads user input from a terminal line by line.
type InputReader struct {
	scanner *bufio.Scanner
}

// NewInputReader creates an InputReader backed by os.Stdin.
func NewInputReader() *InputReader {
	return &InputReader{scanner: bufio.NewScanner(os.Stdin)}
}

// NewInputReaderFrom creates an InputReader from an arbitrary reader (useful for testing).
func NewInputReaderFrom(r io.Reader) *InputReader {
	return &InputReader{scanner: bufio.NewScanner(r)}
}

// ReadLine displays prompt and reads a single trimmed line.
// Returns io.EOF when the input stream is closed (Ctrl+D).
func (ir *InputReader) ReadLine(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, prompt)

	if !ir.scanner.Scan() {
		if err := ir.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(ir.scanner.Text()), nil
}

// ReadMultiLine reads lines until an empty line is entered.
// The trailing blank line is not included in the result.
// Returns io.EOF when the input stream is closed (Ctrl+D).
func (ir *InputReader) ReadMultiLine() (string, error) {
	var lines []string
	for {
		if !ir.scanner.Scan() {
			if err := ir.scanner.Err(); err != nil {
				return strings.Join(lines, "\n"), err
			}
			if len(lines) > 0 {
				return strings.Join(lines, "\n"), nil
			}
			return "", io.EOF
		}
		line := ir.scanner.Text()
		if strings.TrimSpace(line) == "" && len(lines) > 0 {
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}
