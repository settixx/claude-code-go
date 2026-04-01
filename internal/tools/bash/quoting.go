package bash

import "strings"

// QuoteExtraction holds different representations of a command with quotes handled.
type QuoteExtraction struct {
	// WithDoubleQuotes preserves double-quoted content, strips single-quoted.
	WithDoubleQuotes string
	// FullyUnquoted removes all quoted content entirely.
	FullyUnquoted string
	// UnquotedKeepQuoteChars removes content between quotes but leaves the quote characters.
	UnquotedKeepQuoteChars string
}

// ExtractQuotedContent parses a shell command and returns representations
// with quoted regions handled. This enables security checks on the
// "structural" parts of a command without false-positiving on quoted literals.
func ExtractQuotedContent(command string) QuoteExtraction {
	var (
		withDQ   strings.Builder
		fullyUQ  strings.Builder
		keepChar strings.Builder
		inSingle bool
		inDouble bool
		escaped  bool
	)

	for i := 0; i < len(command); i++ {
		ch := command[i]

		if escaped {
			escaped = false
			if !inSingle && !inDouble {
				withDQ.WriteByte(ch)
				fullyUQ.WriteByte(ch)
				keepChar.WriteByte(ch)
			} else if inDouble {
				withDQ.WriteByte(ch)
			}
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			if !inDouble {
				withDQ.WriteByte(ch)
				fullyUQ.WriteByte(ch)
				keepChar.WriteByte(ch)
			}
			continue
		}

		if ch == '\'' && !inDouble {
			if inSingle {
				inSingle = false
				keepChar.WriteByte(ch)
				continue
			}
			inSingle = true
			keepChar.WriteByte(ch)
			continue
		}

		if ch == '"' && !inSingle {
			if inDouble {
				inDouble = false
				withDQ.WriteByte(ch)
				keepChar.WriteByte(ch)
				continue
			}
			inDouble = true
			withDQ.WriteByte(ch)
			keepChar.WriteByte(ch)
			continue
		}

		if inSingle {
			continue
		}
		if inDouble {
			withDQ.WriteByte(ch)
			continue
		}

		withDQ.WriteByte(ch)
		fullyUQ.WriteByte(ch)
		keepChar.WriteByte(ch)
	}

	return QuoteExtraction{
		WithDoubleQuotes:       withDQ.String(),
		FullyUnquoted:          fullyUQ.String(),
		UnquotedKeepQuoteChars: keepChar.String(),
	}
}

// StripSafeRedirections removes common harmless redirections (stderr to stdout,
// to /dev/null, from /dev/null) so they don't trigger redirection checks.
func StripSafeRedirections(content string) string {
	safe := []string{
		"2>&1",
		">/dev/null",
		"2>/dev/null",
		"</dev/null",
		"&>/dev/null",
		"1>/dev/null",
	}
	result := content
	for _, s := range safe {
		result = strings.ReplaceAll(result, s, "")
	}
	return result
}

// HasUnescapedChar checks whether content contains at least one occurrence of
// the target byte that is not preceded by a backslash.
func HasUnescapedChar(content string, ch byte) bool {
	for i := 0; i < len(content); i++ {
		if content[i] != ch {
			continue
		}
		if i == 0 || content[i-1] != '\\' {
			return true
		}
	}
	return false
}

// ExtractBaseCommand returns the first simple command from a pipeline or
// command list. It splits on |, &&, ||, ; and returns the trimmed first token.
func ExtractBaseCommand(command string) string {
	cmd := strings.TrimSpace(command)

	for _, prefix := range []string{"env ", "sudo ", "nohup ", "nice ", "time "} {
		cmd = strings.TrimPrefix(cmd, prefix)
	}

	for _, sep := range []string{"|", "&&", "||", ";"} {
		if idx := strings.Index(cmd, sep); idx >= 0 {
			cmd = cmd[:idx]
		}
	}

	cmd = strings.TrimSpace(cmd)
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
