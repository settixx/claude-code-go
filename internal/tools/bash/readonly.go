package bash

import "strings"

// ValidateReadOnly checks whether a command is safe to run in read-only mode.
// Returns allow for known-safe read commands, deny for anything else.
func ValidateReadOnly(command string) SecurityResult {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return allow("empty command")
	}

	base := ExtractBaseCommand(trimmed)
	if base == "" {
		return deny("could not determine base command for read-only validation")
	}

	spec, ok := readOnlyCommands[base]
	if !ok {
		return deny("command '" + base + "' is not on the read-only allow list")
	}

	if spec.fullAllow {
		return allow("'" + base + "' is unconditionally safe in read-only mode")
	}

	if spec.subcommandCheck != nil {
		return spec.subcommandCheck(trimmed)
	}

	if spec.forbiddenFlags != nil {
		return checkForbiddenFlags(trimmed, base, spec.forbiddenFlags)
	}

	return allow("'" + base + "' passes read-only checks")
}

type readOnlySpec struct {
	fullAllow       bool
	forbiddenFlags  []string
	subcommandCheck func(command string) SecurityResult
}

var readOnlyCommands = map[string]readOnlySpec{
	// Filesystem read ops
	"ls":     {fullAllow: true},
	"cat":    {fullAllow: true},
	"head":   {fullAllow: true},
	"tail":   {fullAllow: true},
	"less":   {fullAllow: true},
	"more":   {fullAllow: true},
	"wc":     {fullAllow: true},
	"stat":   {fullAllow: true},
	"file":   {fullAllow: true},
	"tree":   {fullAllow: true},
	"du":     {fullAllow: true},
	"df":     {fullAllow: true},
	"pwd":    {fullAllow: true},
	"which":  {fullAllow: true},
	"whoami": {fullAllow: true},
	"uname":  {fullAllow: true},
	"echo":   {fullAllow: true},
	"printf": {fullAllow: true},
	"date":   {fullAllow: true},
	"env":    {fullAllow: true},
	"printenv": {fullAllow: true},
	"id":     {fullAllow: true},

	// Search
	"find":   {fullAllow: true},
	"grep":   {fullAllow: true},
	"egrep":  {fullAllow: true},
	"fgrep":  {fullAllow: true},
	"rg":     {forbiddenFlags: rgForbiddenFlags},
	"ag":     {fullAllow: true},
	"fd":     {fullAllow: true},

	// Git (subcommand-based)
	"git": {subcommandCheck: validateGitReadOnly},

	// Docker (subcommand-based)
	"docker": {subcommandCheck: validateDockerReadOnly},

	// GitHub CLI (subcommand-based)
	"gh": {subcommandCheck: validateGhReadOnly},

	// Python tools
	"python":  {forbiddenFlags: pythonForbiddenFlags},
	"python3": {forbiddenFlags: pythonForbiddenFlags},
	"pyright": {fullAllow: true},
	"mypy":    {fullAllow: true},
	"ruff":    {subcommandCheck: validateRuffReadOnly},

	// Node tools
	"node":    {forbiddenFlags: nodeForbiddenFlags},
	"npm":     {subcommandCheck: validateNpmReadOnly},
	"npx":     {fullAllow: false, subcommandCheck: validateNpxReadOnly},
	"tsc":     {forbiddenFlags: tscForbiddenFlags},
	"eslint":  {forbiddenFlags: eslintForbiddenFlags},

	// Build / system inspection
	"make": {subcommandCheck: validateMakeReadOnly},
	"go":   {subcommandCheck: validateGoReadOnly},
	"cargo": {subcommandCheck: validateCargoReadOnly},
	"rustc": {forbiddenFlags: []string{"--emit"}},

	// Misc read-only
	"jq":     {fullAllow: true},
	"yq":     {fullAllow: true},
	"curl":   {forbiddenFlags: curlForbiddenFlags},
	"wget":   {forbiddenFlags: wgetForbiddenFlags},
	"diff":   {fullAllow: true},
	"md5sum": {fullAllow: true},
	"sha256sum": {fullAllow: true},
	"sort":   {fullAllow: true},
	"uniq":   {fullAllow: true},
	"cut":    {fullAllow: true},
	"awk":    {fullAllow: true},
	"sed":    {forbiddenFlags: sedForbiddenFlags},
	"tr":     {fullAllow: true},
	"tee":    {fullAllow: false, subcommandCheck: func(_ string) SecurityResult { return deny("'tee' writes to files") }},
	"xargs":  {fullAllow: false, subcommandCheck: func(_ string) SecurityResult { return deny("'xargs' can execute arbitrary commands") }},
}

// ---------------------------------------------------------------------------
// Forbidden flag lists
// ---------------------------------------------------------------------------

var rgForbiddenFlags = []string{
	"--replace", "-r",
	"--passthru",
}

var pythonForbiddenFlags = []string{"-c", "-m"}
var nodeForbiddenFlags = []string{"-e", "--eval", "-p", "--print"}
var tscForbiddenFlags = []string{"--build", "-b"}
var eslintForbiddenFlags = []string{"--fix", "--fix-dry-run"}
var curlForbiddenFlags = []string{
	"-X", "--request",
	"-d", "--data", "--data-raw", "--data-binary", "--data-urlencode",
	"-F", "--form",
	"-T", "--upload-file",
	"--output", "-o",
	"-O", "--remote-name",
}
var wgetForbiddenFlags = []string{"-O", "--output-document", "-P", "--directory-prefix"}
var sedForbiddenFlags = []string{"-i", "--in-place"}

// ---------------------------------------------------------------------------
// Subcommand validators
// ---------------------------------------------------------------------------

var gitReadOnlySubs = map[string]bool{
	"status": true, "log": true, "diff": true, "show": true,
	"branch": true, "tag": true, "remote": true,
	"rev-parse": true, "rev-list": true, "ls-files": true,
	"ls-tree": true, "ls-remote": true, "cat-file": true,
	"describe": true, "shortlog": true, "blame": true,
	"name-rev": true, "config": true, "stash": true,
	"reflog": true, "grep": true, "for-each-ref": true,
	"symbolic-ref": true, "verify-commit": true,
}

var gitConfigReadFlags = []string{"--get", "--list", "--get-all", "--get-regexp"}

func validateGitReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "git")
	if sub == "" {
		return allow("bare 'git' with no subcommand")
	}
	if !gitReadOnlySubs[sub] {
		return deny("'git " + sub + "' may modify repository state")
	}
	if sub == "config" && isGitConfigWrite(command) {
		return deny("'git config' write detected")
	}
	if sub == "stash" && !isGitStashReadOnly(command) {
		return deny("'git stash' write operation detected")
	}
	return allow("'git " + sub + "' is read-only safe")
}

func isGitConfigWrite(command string) bool {
	for _, flag := range gitConfigReadFlags {
		if strings.Contains(command, flag) {
			return false
		}
	}
	fields := strings.Fields(command)
	afterConfig := false
	nonFlagCount := 0
	for _, f := range fields {
		if !afterConfig {
			if f == "config" {
				afterConfig = true
			}
			continue
		}
		if strings.HasPrefix(f, "-") {
			continue
		}
		nonFlagCount++
	}
	return nonFlagCount >= 2
}

func isGitStashReadOnly(command string) bool {
	for _, safe := range []string{"list", "show"} {
		if strings.Contains(command, "stash "+safe) {
			return true
		}
	}
	return strings.HasSuffix(strings.TrimSpace(command), "git stash")
}

var dockerReadOnlySubs = map[string]bool{
	"ps": true, "images": true, "inspect": true,
	"logs": true, "stats": true, "top": true,
	"port": true, "diff": true, "events": true,
	"version": true, "info": true, "history": true,
	"search": true, "network": true, "volume": true,
}

func validateDockerReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "docker")
	if sub == "" {
		return deny("bare 'docker' without subcommand")
	}
	if dockerReadOnlySubs[sub] {
		return allow("'docker " + sub + "' is read-only safe")
	}
	return deny("'docker " + sub + "' may modify container state")
}

var ghReadOnlySubs = map[string]bool{
	"pr":    true,
	"issue": true,
	"repo":  true,
	"run":   true,
	"api":   true,
}

var ghWriteSubSubs = map[string]map[string]bool{
	"pr": {
		"create": true, "merge": true, "close": true, "reopen": true,
		"edit": true, "ready": true, "review": true, "comment": true,
	},
	"issue": {
		"create": true, "close": true, "reopen": true,
		"edit": true, "delete": true, "comment": true,
		"transfer": true, "pin": true, "unpin": true,
	},
	"repo": {
		"create": true, "delete": true, "fork": true,
		"rename": true, "edit": true, "archive": true,
	},
	"run": {
		"cancel": true, "rerun": true, "delete": true,
	},
}

func validateGhReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "gh")
	if sub == "" {
		return deny("bare 'gh' without subcommand")
	}
	if !ghReadOnlySubs[sub] {
		return deny("'gh " + sub + "' is not on the read-only allow list")
	}

	writeMap, hasWriteMap := ghWriteSubSubs[sub]
	if !hasWriteMap {
		return allow("'gh " + sub + "' is read-only safe")
	}

	subSub := extractSubSubcommand(command, "gh", sub)
	if subSub == "" {
		return allow("'gh " + sub + "' bare invocation")
	}
	if writeMap[subSub] {
		return deny("'gh " + sub + " " + subSub + "' modifies remote state")
	}
	return allow("'gh " + sub + " " + subSub + "' is read-only safe")
}

func validateRuffReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "ruff")
	if sub == "check" || sub == "format" {
		if strings.Contains(command, "--fix") || strings.Contains(command, "--diff") {
			return deny("'ruff " + sub + "' with --fix or --diff modifies files")
		}
		return allow("'ruff " + sub + "' in check-only mode")
	}
	return deny("'ruff " + sub + "' is not on the read-only allow list")
}

func validateNpmReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "npm")
	safeSubs := map[string]bool{
		"ls": true, "list": true, "info": true, "view": true,
		"search": true, "outdated": true, "audit": true, "explain": true,
		"why": true, "config": true, "version": true, "help": true,
	}
	if safeSubs[sub] {
		return allow("'npm " + sub + "' is read-only safe")
	}
	return deny("'npm " + sub + "' may modify node_modules or package.json")
}

func validateNpxReadOnly(command string) SecurityResult {
	return deny("'npx' can execute arbitrary packages")
}

func validateMakeReadOnly(command string) SecurityResult {
	if strings.Contains(command, "-n") || strings.Contains(command, "--dry-run") || strings.Contains(command, "--just-print") {
		return allow("'make' in dry-run mode")
	}
	return deny("'make' without -n/--dry-run may execute build steps")
}

func validateGoReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "go")
	safeSubs := map[string]bool{
		"version": true, "env": true, "list": true,
		"doc": true, "vet": true, "help": true,
	}
	if safeSubs[sub] {
		return allow("'go " + sub + "' is read-only safe")
	}
	return deny("'go " + sub + "' may modify files or build artifacts")
}

func validateCargoReadOnly(command string) SecurityResult {
	sub := extractSubcommand(command, "cargo")
	safeSubs := map[string]bool{
		"check": true, "clippy": true, "doc": true,
		"tree": true, "metadata": true, "version": true,
		"search": true, "info": true, "verify-project": true,
	}
	if safeSubs[sub] {
		return allow("'cargo " + sub + "' is read-only safe")
	}
	return deny("'cargo " + sub + "' may modify files or build artifacts")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func checkForbiddenFlags(command, base string, forbidden []string) SecurityResult {
	fields := strings.Fields(command)
	for _, f := range fields {
		for _, bad := range forbidden {
			if f == bad || strings.HasPrefix(f, bad+"=") {
				return deny("'" + base + "' with flag '" + bad + "' is not allowed in read-only mode")
			}
		}
	}
	return allow("'" + base + "' passes read-only flag checks")
}

func extractSubcommand(command, program string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	foundProgram := false
	for _, f := range fields {
		if !foundProgram {
			if f == program {
				foundProgram = true
			}
			continue
		}
		if strings.HasPrefix(f, "-") {
			continue
		}
		return f
	}
	return ""
}

func extractSubSubcommand(command, program, sub string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	foundProgram := false
	foundSub := false
	for _, f := range fields {
		if !foundProgram {
			if f == program {
				foundProgram = true
			}
			continue
		}
		if !foundSub {
			if f == sub {
				foundSub = true
			}
			continue
		}
		if strings.HasPrefix(f, "-") {
			continue
		}
		return f
	}
	return ""
}
