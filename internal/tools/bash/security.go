package bash

import (
	"encoding/base64"
	"regexp"
	"strings"
	"unicode"
)

// SecurityBehavior describes the outcome of a security check.
type SecurityBehavior string

const (
	SecurityAllow       SecurityBehavior = "allow"
	SecurityDeny        SecurityBehavior = "deny"
	SecurityAsk         SecurityBehavior = "ask"
	SecurityPassthrough SecurityBehavior = "passthrough"
)

// SecurityResult is returned by each validator in the chain.
type SecurityResult struct {
	Behavior SecurityBehavior
	Message  string
}

func allow(msg string) SecurityResult  { return SecurityResult{SecurityAllow, msg} }
func deny(msg string) SecurityResult   { return SecurityResult{SecurityDeny, msg} }
func ask(msg string) SecurityResult    { return SecurityResult{SecurityAsk, msg} }
func pass() SecurityResult             { return SecurityResult{SecurityPassthrough, ""} }

// Validator is a single step in the security chain.
type Validator func(command string) SecurityResult

// ValidateCommand runs the full security validator chain against a command.
// The first non-passthrough result wins. If all validators pass through,
// the command is sent for user confirmation (ask).
func ValidateCommand(command string) SecurityResult {
	for _, v := range securityChain {
		result := v(command)
		if result.Behavior != SecurityPassthrough {
			return result
		}
	}
	return ask("command requires approval")
}

// securityChain is the ordered list of validators. First allow/deny wins.
var securityChain = []Validator{
	validateEmpty,
	validateIncompleteCommands,
	validateNewlines,
	validateCommandSubstitution,
	validateZshDangerousCommands,
	validateHeredocInjection,
	validateShellQuoteMalformation,
	validateDangerousPatterns,
	validateObfuscatedFlags,
	validateShellMetacharacters,
	validateDangerousVariables,
}

// ---------------------------------------------------------------------------
// Validators
// ---------------------------------------------------------------------------

func validateEmpty(command string) SecurityResult {
	if strings.TrimSpace(command) == "" {
		return allow("empty command")
	}
	return pass()
}

func validateIncompleteCommands(command string) SecurityResult {
	trimmed := strings.TrimSpace(command)
	if len(trimmed) == 0 {
		return pass()
	}

	if trimmed[0] == '\t' {
		return deny("command begins with a tab — likely a paste error")
	}

	for _, prefix := range incompleteCommandPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return deny("command begins with an operator or flag fragment: " + prefix)
		}
	}
	return pass()
}

var incompleteCommandPrefixes = []string{
	"&&", "||", ";", ">>", ">", "| ", "-",
}

func validateNewlines(command string) SecurityResult {
	if strings.ContainsAny(command, "\n\r") {
		return deny("command contains newline characters — potential injection vector")
	}
	return pass()
}

func validateCommandSubstitution(command string) SecurityResult {
	extracted := ExtractQuotedContent(command)
	raw := extracted.FullyUnquoted

	for _, pat := range cmdSubPatterns {
		if pat.MatchString(raw) {
			return deny("command contains command substitution or expansion syntax")
		}
	}

	for _, lit := range cmdSubLiterals {
		if strings.Contains(raw, lit) {
			return deny("command contains command substitution or expansion syntax")
		}
	}

	if strings.Contains(raw, "`") {
		return deny("command contains backtick command substitution")
	}

	return pass()
}

var cmdSubPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\$\(`),
	regexp.MustCompile(`\$\{`),
	regexp.MustCompile(`\$\[`),
	regexp.MustCompile(`<\(`),
	regexp.MustCompile(`>\(`),
	regexp.MustCompile(`=\(`),
}

var cmdSubLiterals = []string{
	"~[",       // zsh dynamic named directory
	"(e:",      // zsh glob qualifier eval
	"(+",       // zsh glob qualifier extended
	"} always {", // zsh always block
	"<#",       // PowerShell block comment
}

// zshDangerousModules lists zsh built-in commands that can escape sandboxes.
var zshDangerousModules = map[string]bool{
	"zmodload": true, "emulate": true,
	"sysopen": true, "sysread": true, "syswrite": true, "sysseek": true,
	"zpty": true, "ztcp": true, "zsocket": true,
	"mapfile": true,
	"zf_rm": true, "zf_mv": true, "zf_ln": true,
	"zf_chmod": true, "zf_chown": true, "zf_mkdir": true,
	"zf_rmdir": true, "zf_chgrp": true,
}

func validateZshDangerousCommands(command string) SecurityResult {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return pass()
	}
	base := fields[0]
	if zshDangerousModules[base] {
		return deny("blocked zsh built-in: " + base)
	}
	return pass()
}

func validateHeredocInjection(command string) SecurityResult {
	if !heredocInCmdSubRe.MatchString(command) {
		return pass()
	}
	return deny("heredoc inside command substitution — potential injection")
}

var heredocInCmdSubRe = regexp.MustCompile(`\$\([^)]*<<`)

func validateShellQuoteMalformation(command string) SecurityResult {
	singles := strings.Count(command, "'") - strings.Count(command, "\\'")
	doubles := strings.Count(command, "\"") - strings.Count(command, "\\\"")

	if singles%2 != 0 {
		return deny("unbalanced single quotes — malformed shell quoting")
	}
	if doubles%2 != 0 {
		return deny("unbalanced double quotes — malformed shell quoting")
	}
	return pass()
}

func validateDangerousPatterns(command string) SecurityResult {
	extracted := ExtractQuotedContent(command)
	unquoted := extracted.FullyUnquoted
	unquotedSafe := StripSafeRedirections(unquoted)

	if HasUnescapedChar(unquoted, '`') {
		return deny("unescaped backtick outside quotes")
	}

	if hasUnsafeRedirection(unquotedSafe) {
		return ask("command contains I/O redirection — confirm intent")
	}

	if ifsRe.MatchString(unquoted) {
		return deny("IFS manipulation can alter command parsing")
	}

	if gitCommitSubRe.MatchString(command) {
		return deny("git commit message with command substitution")
	}

	if strings.Contains(command, "/proc/") && strings.Contains(command, "environ") {
		return deny("reading /proc/environ leaks secrets")
	}

	if containsControlChars(command) {
		return deny("command contains control characters")
	}

	if containsUnicodeWhitespace(command) {
		return deny("command contains non-ASCII whitespace (possible obfuscation)")
	}

	if braceExpansionRe.MatchString(unquoted) {
		return ask("command uses brace expansion")
	}

	if midWordHashRe.MatchString(unquoted) {
		return ask("command contains mid-word # (ambiguous intent)")
	}

	for _, dp := range destructivePatterns {
		if strings.Contains(strings.ToLower(strings.TrimSpace(command)), dp) {
			return deny("blocked dangerous pattern: " + dp)
		}
	}

	return pass()
}

var destructivePatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs.",
	"dd if=/dev/zero",
	"dd if=/dev/random",
	":(){:|:&};:",
	"> /dev/sda",
	"chmod -R 777 /",
}

var (
	ifsRe            = regexp.MustCompile(`\bIFS\s*=`)
	gitCommitSubRe   = regexp.MustCompile(`git\s+commit\s+.*\$\(`)
	braceExpansionRe = regexp.MustCompile(`\{[^}]*,[^}]*\}`)
	midWordHashRe    = regexp.MustCompile(`\w#\w`)
)

func hasUnsafeRedirection(content string) bool {
	for _, ch := range []byte{'>', '<'} {
		if HasUnescapedChar(content, ch) {
			return true
		}
	}
	return false
}

func containsControlChars(s string) bool {
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
		if r == 0x7F {
			return true
		}
	}
	return false
}

func containsUnicodeWhitespace(s string) bool {
	for _, r := range s {
		if r > 127 && unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func validateObfuscatedFlags(command string) SecurityResult {
	fields := strings.Fields(command)
	for _, f := range fields {
		if !strings.HasPrefix(f, "-") {
			continue
		}
		if looksBase64Encoded(f) {
			return deny("flag appears base64-encoded — possible obfuscation")
		}
		if looksHexEncoded(f) {
			return deny("flag appears hex-encoded — possible obfuscation")
		}
	}
	return pass()
}

func looksBase64Encoded(s string) bool {
	payload := strings.TrimLeft(s, "-")
	if len(payload) < 8 {
		return false
	}
	if !base64ContentRe.MatchString(payload) {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
	}
	if err != nil {
		return false
	}
	return len(decoded) > 4 && isPrintable(decoded)
}

var base64ContentRe = regexp.MustCompile(`^[A-Za-z0-9+/=]{8,}$`)

func looksHexEncoded(s string) bool {
	payload := strings.TrimLeft(s, "-")
	if len(payload) < 8 || len(payload)%2 != 0 {
		return false
	}
	return hexContentRe.MatchString(payload)
}

var hexContentRe = regexp.MustCompile(`^[0-9a-fA-F]+$`)

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7E {
			return false
		}
	}
	return true
}

func validateShellMetacharacters(command string) SecurityResult {
	extracted := ExtractQuotedContent(command)
	raw := extracted.FullyUnquoted

	for _, mc := range dangerousMetacharRe {
		if mc.MatchString(raw) {
			return ask("command contains shell metacharacter that may alter execution")
		}
	}
	return pass()
}

var dangerousMetacharRe = []*regexp.Regexp{
	regexp.MustCompile(`;\s*\w`),      // sequential execution after semicolon
	regexp.MustCompile(`\|\s*\w`),     // pipe into another command
	regexp.MustCompile(`&&\s*\w`),     // conditional chain
	regexp.MustCompile(`\|\|\s*\w`),   // or-chain
}

// dangerousShellVars are environment variables that, if set by a command,
// can hijack subsequent shell execution.
var dangerousShellVars = []string{
	"BASH_ENV",
	"ENV",
	"PROMPT_COMMAND",
	"BASH_FUNC_",
	"SHELLOPTS",
	"BASHOPTS",
	"GLOBIGNORE",
	"BASH_XTRACEFD",
	"HISTFILE",
	"HISTCONTROL",
	"CDPATH",
	"LD_PRELOAD",
	"LD_LIBRARY_PATH",
	"DYLD_INSERT_LIBRARIES",
	"DYLD_LIBRARY_PATH",
}

func validateDangerousVariables(command string) SecurityResult {
	upper := strings.ToUpper(command)
	for _, v := range dangerousShellVars {
		pattern := v + "="
		if !strings.Contains(upper, strings.ToUpper(pattern)) {
			continue
		}
		if strings.Contains(command, pattern) || strings.Contains(command, strings.ToLower(pattern)) {
			return deny("command sets dangerous shell variable: " + v)
		}
	}

	if strings.Contains(command, "export ") {
		for _, v := range dangerousShellVars {
			exportPat := "export " + v
			if strings.Contains(command, exportPat) {
				return deny("command exports dangerous shell variable: " + v)
			}
		}
	}
	return pass()
}
