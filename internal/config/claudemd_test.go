package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// stripHTMLComments
// ---------------------------------------------------------------------------

func TestStripHTMLComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no comments", "hello world", "hello world"},
		{"single comment", "before <!-- hidden --> after", "before  after"},
		{"multi-line comment", "a\n<!-- foo\nbar\nbaz -->\nb", "a\n\nb"},
		{"multiple comments", "a <!-- 1 --> b <!-- 2 --> c", "a  b  c"},
		{"empty comment", "x <!-- --> y", "x  y"},
		{"nested arrows", "<!-- <b> --> text", " text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTMLComments(tt.input)
			if got != tt.want {
				t.Errorf("stripHTMLComments(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// classifyHeading / extractListItem
// ---------------------------------------------------------------------------

func TestClassifyHeading(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# allowed tools", "allow"},
		{"# allow tools", "allow"},
		{"## denied tools", "deny"},
		{"### deny list", "deny"},
		{"# random heading", ""},
		{"# my allowed commands", "allow"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := classifyHeading(strings.ToLower(tt.input))
			if got != tt.want {
				t.Errorf("classifyHeading(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractListItem(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"- Bash", "Bash"},
		{"* FileRead", "FileRead"},
		{"- `Bash(git *)`", "Bash(git *)"},
		{"plain text", ""},
		{"  - indented", ""},
		{"- ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractListItem(tt.input)
			if got != tt.want {
				t.Errorf("extractListItem(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseClaudeMDContent
// ---------------------------------------------------------------------------

func TestParseClaudeMDContent(t *testing.T) {
	content := `# Project Guidelines
Some instructions here.

# Allowed tools
- Bash
- FileRead

# Denied tools
- PowerShell
- WebFetch

# Other section
- SomeItem
`
	rules := parseClaudeMDContent(content)

	wantAllow := []string{"Bash", "FileRead"}
	wantDeny := []string{"PowerShell", "WebFetch"}

	if len(rules.AllowPatterns) != len(wantAllow) {
		t.Fatalf("AllowPatterns len = %d, want %d", len(rules.AllowPatterns), len(wantAllow))
	}
	for i, p := range rules.AllowPatterns {
		if p != wantAllow[i] {
			t.Errorf("AllowPatterns[%d] = %q, want %q", i, p, wantAllow[i])
		}
	}

	if len(rules.DenyPatterns) != len(wantDeny) {
		t.Fatalf("DenyPatterns len = %d, want %d", len(rules.DenyPatterns), len(wantDeny))
	}
	for i, p := range rules.DenyPatterns {
		if p != wantDeny[i] {
			t.Errorf("DenyPatterns[%d] = %q, want %q", i, p, wantDeny[i])
		}
	}

	if rules.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestParseClaudeMDContent_EmptyInput(t *testing.T) {
	rules := parseClaudeMDContent("")
	if len(rules.AllowPatterns) != 0 {
		t.Errorf("expected no allow patterns, got %d", len(rules.AllowPatterns))
	}
	if len(rules.DenyPatterns) != 0 {
		t.Errorf("expected no deny patterns, got %d", len(rules.DenyPatterns))
	}
}

// ---------------------------------------------------------------------------
// resolveIncludePath
// ---------------------------------------------------------------------------

func TestResolveIncludePath(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		baseDir string
		wantRel bool
	}{
		{"relative path", "rules.md", "/base/dir", true},
		{"quoted path", `"rules.md"`, "/base/dir", true},
		{"empty path", "", "/base/dir", false},
		{"single-quoted path", `'extra.md'`, "/base/dir", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveIncludePath(tt.raw, tt.baseDir)
			if tt.wantRel && got == "" {
				t.Errorf("resolveIncludePath(%q, %q) returned empty, wanted non-empty", tt.raw, tt.baseDir)
			}
			if !tt.wantRel && got != "" {
				t.Errorf("resolveIncludePath(%q, %q) = %q, wanted empty", tt.raw, tt.baseDir, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveIncludes — @include directive processing
// ---------------------------------------------------------------------------

func TestResolveIncludes_Basic(t *testing.T) {
	tmp := t.TempDir()
	included := filepath.Join(tmp, "extra.md")
	os.WriteFile(included, []byte("included content"), 0o644)

	content := "before\n@include extra.md\nafter"
	visited := map[string]bool{}

	result := resolveIncludes(content, tmp, 0, visited)

	if !strings.Contains(result, "included content") {
		t.Errorf("result should contain 'included content', got %q", result)
	}
	if !strings.Contains(result, "before") {
		t.Errorf("result should contain 'before', got %q", result)
	}
	if !strings.Contains(result, "after") {
		t.Errorf("result should contain 'after', got %q", result)
	}
}

func TestResolveIncludes_CycleDetection(t *testing.T) {
	tmp := t.TempDir()
	fileA := filepath.Join(tmp, "a.md")
	fileB := filepath.Join(tmp, "b.md")

	os.WriteFile(fileA, []byte("A\n@include b.md"), 0o644)
	os.WriteFile(fileB, []byte("B\n@include a.md"), 0o644)

	visited := map[string]bool{}
	absA, _ := filepath.Abs(fileA)
	visited[absA] = true
	result := resolveIncludes("@include b.md", tmp, 0, visited)

	if strings.Count(result, "A") > 1 {
		t.Error("cycle detection failed — A included multiple times")
	}
}

func TestResolveIncludes_DepthLimit(t *testing.T) {
	tmp := t.TempDir()
	deepFile := filepath.Join(tmp, "deep.md")
	os.WriteFile(deepFile, []byte("deep content"), 0o644)

	content := "@include deep.md"
	visited := map[string]bool{}
	result := resolveIncludes(content, tmp, maxIncludeDepth, visited)

	if strings.Contains(result, "deep content") {
		t.Error("should not include content at max depth")
	}
}

func TestResolveIncludes_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	content := "before\n@include nonexistent.md\nafter"
	visited := map[string]bool{}

	result := resolveIncludes(content, tmp, 0, visited)

	if !strings.Contains(result, "before") {
		t.Errorf("result should contain 'before', got %q", result)
	}
	if !strings.Contains(result, "after") {
		t.Errorf("result should contain 'after', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// collectMDsInDir
// ---------------------------------------------------------------------------

func TestCollectMDsInDir(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.md"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(tmp, "b.MD"), []byte("b"), 0o644)
	os.WriteFile(filepath.Join(tmp, "c.txt"), []byte("c"), 0o644)
	os.Mkdir(filepath.Join(tmp, "subdir.md"), 0o755)

	paths := collectMDsInDir(tmp)
	if len(paths) != 2 {
		t.Errorf("expected 2 .md files, got %d: %v", len(paths), paths)
	}
}

func TestCollectMDsInDir_NonexistentDir(t *testing.T) {
	paths := collectMDsInDir("/nonexistent/dir/that/should/not/exist")
	if len(paths) != 0 {
		t.Errorf("expected empty, got %v", paths)
	}
}

// ---------------------------------------------------------------------------
// collectClaudeMDPaths
// ---------------------------------------------------------------------------

func TestCollectClaudeMDPaths(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub", "project")
	os.MkdirAll(sub, 0o755)

	os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("root"), 0o644)
	os.WriteFile(filepath.Join(sub, "CLAUDE.md"), []byte("sub"), 0o644)

	paths := collectClaudeMDPaths(sub, root)

	if len(paths) < 2 {
		t.Fatalf("expected at least 2 paths, got %d: %v", len(paths), paths)
	}
	if !strings.HasSuffix(paths[0], filepath.Join("project", "CLAUDE.md")) {
		t.Errorf("first path should be deepest, got %q", paths[0])
	}
}

// ---------------------------------------------------------------------------
// parseClaudeMDFile (integration with frontmatter & includes)
// ---------------------------------------------------------------------------

func TestParseClaudeMDFile_WithFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	content := `---
- /some/specific/path
---
# Allowed tools
- Bash
`
	mdPath := filepath.Join(tmp, "CLAUDE.md")
	os.WriteFile(mdPath, []byte(content), 0o644)

	rules, err := parseClaudeMDFile(mdPath)
	if err != nil {
		t.Fatalf("parseClaudeMDFile: %v", err)
	}

	if len(rules.AllowPatterns) != 0 {
		t.Logf("frontmatter path filter did not match (expected), AllowPatterns: %v", rules.AllowPatterns)
	}
}

func TestParseClaudeMDFile_WithIncludes(t *testing.T) {
	tmp := t.TempDir()
	extra := filepath.Join(tmp, "extra-rules.md")
	os.WriteFile(extra, []byte("# Denied tools\n- WebFetch\n"), 0o644)

	main := filepath.Join(tmp, "CLAUDE.md")
	os.WriteFile(main, []byte("# Allowed tools\n- Bash\n@include extra-rules.md\n"), 0o644)

	rules, err := parseClaudeMDFile(main)
	if err != nil {
		t.Fatalf("parseClaudeMDFile: %v", err)
	}

	if len(rules.AllowPatterns) != 1 || rules.AllowPatterns[0] != "Bash" {
		t.Errorf("AllowPatterns = %v, want [Bash]", rules.AllowPatterns)
	}
	if len(rules.DenyPatterns) != 1 || rules.DenyPatterns[0] != "WebFetch" {
		t.Errorf("DenyPatterns = %v, want [WebFetch]", rules.DenyPatterns)
	}
}

// ---------------------------------------------------------------------------
// LoadClaudeMD (multi-scope integration)
// ---------------------------------------------------------------------------

func TestLoadClaudeMD_MultiScope(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "myproject")
	os.MkdirAll(projectDir, 0o755)

	// Without a git repo, findGitRoot returns startDir itself,
	// so the walk only collects CLAUDE.md from projectDir downward.
	// Only the project-level CLAUDE.md will be found.
	os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# Allowed tools\n- FileRead\n"), 0o644)
	os.WriteFile(filepath.Join(projectDir, "CLAUDE.md"), []byte("# Denied tools\n- Bash\n"), 0o644)

	rules, err := LoadClaudeMD(projectDir)
	if err != nil {
		t.Fatalf("LoadClaudeMD: %v", err)
	}

	if rules == nil {
		t.Fatal("rules should not be nil")
	}

	hasBash := false
	for _, p := range rules.DenyPatterns {
		if p == "Bash" {
			hasBash = true
		}
	}

	if !hasBash {
		t.Errorf("expected Bash in DenyPatterns, got %v", rules.DenyPatterns)
	}
}

func TestLoadClaudeMD_StripsHTMLComments(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("visible <!-- hidden --> text"), 0o644)

	rules, err := LoadClaudeMD(dir)
	if err != nil {
		t.Fatalf("LoadClaudeMD: %v", err)
	}

	if strings.Contains(rules.Content, "hidden") {
		t.Errorf("Content should not contain HTML comments, got %q", rules.Content)
	}
	if !strings.Contains(rules.Content, "visible") {
		t.Errorf("Content should contain 'visible', got %q", rules.Content)
	}
}

func TestLoadClaudeMD_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	rules, err := LoadClaudeMD(dir)
	if err != nil {
		t.Fatalf("LoadClaudeMD: %v", err)
	}
	if rules == nil {
		t.Fatal("rules should not be nil even for empty dir")
	}
	if len(rules.AllowPatterns) != 0 || len(rules.DenyPatterns) != 0 {
		t.Errorf("expected empty patterns for dir with no CLAUDE.md")
	}
}

func TestLoadClaudeMD_LocalOverride(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.local.md"), []byte("# Allowed tools\n- LocalTool\n"), 0o644)

	rules, err := LoadClaudeMD(dir)
	if err != nil {
		t.Fatalf("LoadClaudeMD: %v", err)
	}

	hasLocalTool := false
	for _, p := range rules.AllowPatterns {
		if p == "LocalTool" {
			hasLocalTool = true
		}
	}
	if !hasLocalTool {
		t.Errorf("expected LocalTool from CLAUDE.local.md, got %v", rules.AllowPatterns)
	}
}

// ---------------------------------------------------------------------------
// LoadClaudeMD — project rules dir
// ---------------------------------------------------------------------------

func TestLoadClaudeMD_ProjectRulesDir(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".claude", "rules")
	os.MkdirAll(rulesDir, 0o755)
	os.WriteFile(filepath.Join(rulesDir, "custom.md"), []byte("# Allowed tools\n- CustomRuleTool\n"), 0o644)

	rules, err := LoadClaudeMD(dir)
	if err != nil {
		t.Fatalf("LoadClaudeMD: %v", err)
	}

	has := false
	for _, p := range rules.AllowPatterns {
		if p == "CustomRuleTool" {
			has = true
		}
	}
	if !has {
		t.Errorf("expected CustomRuleTool from .claude/rules/, got %v", rules.AllowPatterns)
	}
}

// ---------------------------------------------------------------------------
// LoadClaudeMD — merge multiple scopes
// ---------------------------------------------------------------------------

func TestLoadClaudeMD_MergesMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Allowed tools\n- MainTool\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "CLAUDE.local.md"), []byte("# Denied tools\n- BadTool\n"), 0o644)

	rules, err := LoadClaudeMD(dir)
	if err != nil {
		t.Fatalf("LoadClaudeMD: %v", err)
	}

	hasMain := false
	for _, p := range rules.AllowPatterns {
		if p == "MainTool" {
			hasMain = true
		}
	}
	hasBad := false
	for _, p := range rules.DenyPatterns {
		if p == "BadTool" {
			hasBad = true
		}
	}
	if !hasMain {
		t.Errorf("expected MainTool in AllowPatterns, got %v", rules.AllowPatterns)
	}
	if !hasBad {
		t.Errorf("expected BadTool in DenyPatterns, got %v", rules.DenyPatterns)
	}
}

// ---------------------------------------------------------------------------
// resolveIncludes — nested includes
// ---------------------------------------------------------------------------

func TestResolveIncludes_NestedIncludes(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.md"), []byte("A content\n@include b.md\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "b.md"), []byte("B content\n@include c.md\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "c.md"), []byte("C content"), 0o644)

	visited := map[string]bool{}
	result := resolveIncludes("@include a.md", tmp, 0, visited)

	for _, want := range []string{"A content", "B content", "C content"} {
		if !strings.Contains(result, want) {
			t.Errorf("result missing %q, got %q", want, result)
		}
	}
}

// ---------------------------------------------------------------------------
// parseClaudeMDContent — compound patterns
// ---------------------------------------------------------------------------

func TestParseClaudeMDContent_CompoundPatterns(t *testing.T) {
	content := `# Allowed tools
- Bash(git *)
- FileWrite(/tmp/*)

# Denied tools
- PowerShell
- Bash(rm *)
`
	rules := parseClaudeMDContent(content)

	wantAllow := []string{"Bash(git *)", "FileWrite(/tmp/*)"}
	wantDeny := []string{"PowerShell", "Bash(rm *)"}

	if len(rules.AllowPatterns) != len(wantAllow) {
		t.Fatalf("AllowPatterns len = %d, want %d", len(rules.AllowPatterns), len(wantAllow))
	}
	for i, p := range rules.AllowPatterns {
		if p != wantAllow[i] {
			t.Errorf("AllowPatterns[%d] = %q, want %q", i, p, wantAllow[i])
		}
	}
	if len(rules.DenyPatterns) != len(wantDeny) {
		t.Fatalf("DenyPatterns len = %d, want %d", len(rules.DenyPatterns), len(wantDeny))
	}
	for i, p := range rules.DenyPatterns {
		if p != wantDeny[i] {
			t.Errorf("DenyPatterns[%d] = %q, want %q", i, p, wantDeny[i])
		}
	}
}

// ---------------------------------------------------------------------------
// parseClaudeMDFile — frontmatter matching cwd
// ---------------------------------------------------------------------------

func TestParseClaudeMDFile_FrontmatterMatchesCwd(t *testing.T) {
	tmp := t.TempDir()
	content := "---\n- " + tmp + "\n---\n# Allowed tools\n- MatchedTool\n"
	mdPath := filepath.Join(tmp, "CLAUDE.md")
	os.WriteFile(mdPath, []byte(content), 0o644)

	rules, err := parseClaudeMDFile(mdPath)
	if err != nil {
		t.Fatalf("parseClaudeMDFile: %v", err)
	}

	if len(rules.AllowPatterns) != 1 || rules.AllowPatterns[0] != "MatchedTool" {
		t.Errorf("expected [MatchedTool], got %v", rules.AllowPatterns)
	}
}

// ---------------------------------------------------------------------------
// stripHTMLComments — edge cases
// ---------------------------------------------------------------------------

func TestStripHTMLComments_AdjacentComments(t *testing.T) {
	got := stripHTMLComments("a<!-- x --><!-- y -->b")
	if got != "ab" {
		t.Errorf("stripHTMLComments = %q, want %q", got, "ab")
	}
}

func TestStripHTMLComments_Unclosed(t *testing.T) {
	input := "before <!-- unclosed"
	got := stripHTMLComments(input)
	if got != input {
		t.Errorf("unclosed comment should be left as-is, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// collectClaudeMDPaths — single dir (no git)
// ---------------------------------------------------------------------------

func TestCollectClaudeMDPaths_SingleDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("x"), 0o644)

	paths := collectClaudeMDPaths(dir, dir)
	if len(paths) != 1 {
		t.Errorf("expected 1 path for single dir, got %d: %v", len(paths), paths)
	}
}

// ---------------------------------------------------------------------------
// resolveIncludePath — absolute path
// ---------------------------------------------------------------------------

func TestResolveIncludePath_AbsolutePath(t *testing.T) {
	got := resolveIncludePath("/absolute/path.md", "/some/base")
	if got != "/absolute/path.md" {
		t.Errorf("resolveIncludePath absolute = %q, want /absolute/path.md", got)
	}
}
