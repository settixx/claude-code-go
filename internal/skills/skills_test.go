package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// YAML frontmatter parsing
// ---------------------------------------------------------------------------

func TestSplitFrontmatter_Present(t *testing.T) {
	raw := "---\nname: test\ndescription: hello\n---\nBody content here"
	fm, body, found := splitFrontmatter(raw)
	if !found {
		t.Fatal("expected frontmatter to be found")
	}
	if !strings.Contains(fm, "name: test") {
		t.Errorf("frontmatter = %q, want to contain 'name: test'", fm)
	}
	if !strings.Contains(body, "Body content here") {
		t.Errorf("body = %q, want to contain 'Body content here'", body)
	}
}

func TestSplitFrontmatter_Missing(t *testing.T) {
	raw := "No frontmatter here"
	_, body, found := splitFrontmatter(raw)
	if found {
		t.Error("should not find frontmatter")
	}
	if body != raw {
		t.Errorf("body should be original text, got %q", body)
	}
}

func TestSplitFrontmatter_UnclosedDelimiter(t *testing.T) {
	raw := "---\nname: test\nno closing delimiter"
	_, _, found := splitFrontmatter(raw)
	if found {
		t.Error("should not find frontmatter with unclosed delimiter")
	}
}

func TestParseFrontmatter_AllFields(t *testing.T) {
	s := &Skill{}
	fm := "name: my-skill\ndescription: does things\ntags: [go, test]\nuser_invocable: true\nauto_run: false"
	parseFrontmatter(fm, s)

	if s.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "my-skill")
	}
	if s.Description != "does things" {
		t.Errorf("Description = %q, want %q", s.Description, "does things")
	}
	if len(s.Tags) != 2 || s.Tags[0] != "go" || s.Tags[1] != "test" {
		t.Errorf("Tags = %v, want [go, test]", s.Tags)
	}
	if !s.UserInvoke {
		t.Error("UserInvoke should be true")
	}
	if s.AutoRun {
		t.Error("AutoRun should be false")
	}
}

func TestParseYAMLList_BracketSyntax(t *testing.T) {
	got := parseYAMLList("[a, b, c]")
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("got %v", got)
	}
}

func TestParseYAMLList_DashSyntax(t *testing.T) {
	got := parseYAMLList("- item")
	if len(got) != 1 || got[0] != "item" {
		t.Errorf("got %v, want [item]", got)
	}
}

func TestParseYAMLList_SingleValue(t *testing.T) {
	got := parseYAMLList("solo")
	if len(got) != 1 || got[0] != "solo" {
		t.Errorf("got %v, want [solo]", got)
	}
}

func TestParseYAMLList_Empty(t *testing.T) {
	got := parseYAMLList("")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestParseSkillContent_FullDocument(t *testing.T) {
	raw := `---
name: my-skill
description: A test skill
tags: [testing]
user_invocable: true
---
# Skill Instructions

Do the thing.`

	s, err := parseSkillContent(raw, "/fake/path/my-skill/SKILL.md")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if s.Name != "my-skill" {
		t.Errorf("Name = %q", s.Name)
	}
	if !strings.Contains(s.Content, "Skill Instructions") {
		t.Error("content should contain body text")
	}
	if !s.UserInvoke {
		t.Error("UserInvoke should be true")
	}
}

func TestParseSkillContent_NoFrontmatter(t *testing.T) {
	raw := "# Just a markdown file\n\nSome content."
	s, err := parseSkillContent(raw, "/fake/skills/my-tool/SKILL.md")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if s.Name != "my-tool" {
		t.Errorf("Name = %q, should be inferred from path", s.Name)
	}
	if !strings.Contains(s.Content, "Just a markdown file") {
		t.Error("content should preserve body")
	}
}

func TestInferNameFromPath(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{"/home/user/.claude/skills/git-commit/SKILL.md", "git-commit"},
		{"/skills/debug/SKILL.md", "debug"},
		{"/SKILL.md", "/"},
	}
	for _, tt := range tests {
		got := inferNameFromPath(tt.path)
		if got != tt.want {
			t.Errorf("inferNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// SkillRegistry
// ---------------------------------------------------------------------------

func TestSkillRegistry_RegisterAndGet(t *testing.T) {
	r := NewSkillRegistry()
	s := &Skill{Name: "test", Description: "a test skill", Content: "do stuff"}
	r.Register(s)

	got, ok := r.Get("test")
	if !ok {
		t.Fatal("skill not found")
	}
	if got.Description != "a test skill" {
		t.Errorf("Description = %q", got.Description)
	}
}

func TestSkillRegistry_OverwriteExisting(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "s1", Content: "v1"})
	r.Register(&Skill{Name: "s1", Content: "v2"})

	got, _ := r.Get("s1")
	if got.Content != "v2" {
		t.Errorf("Content = %q, want %q (should overwrite)", got.Content, "v2")
	}
}

func TestSkillRegistry_GetNotFound(t *testing.T) {
	r := NewSkillRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent skill")
	}
}

func TestSkillRegistry_All(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "a"})
	r.Register(&Skill{Name: "b"})
	r.Register(&Skill{Name: "c"})

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d skills, want 3", len(all))
	}
}

func TestSkillRegistry_UserInvocable(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "pub", UserInvoke: true})
	r.Register(&Skill{Name: "priv", UserInvoke: false})
	r.Register(&Skill{Name: "pub2", UserInvoke: true})

	inv := r.UserInvocable()
	if len(inv) != 2 {
		t.Errorf("UserInvocable() returned %d, want 2", len(inv))
	}
}

func TestSkillRegistry_FindByTag(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "s1", Tags: []string{"go", "test"}})
	r.Register(&Skill{Name: "s2", Tags: []string{"python"}})
	r.Register(&Skill{Name: "s3", Tags: []string{"go", "debug"}})

	got := r.FindByTag("go")
	if len(got) != 2 {
		t.Errorf("FindByTag('go') returned %d, want 2", len(got))
	}
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

func TestSkillSearch_MatchesByName(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "git-commit", Description: "commit changes"})
	r.Register(&Skill{Name: "debug", Description: "debug code"})

	results := r.Search("git")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "git-commit" {
		t.Errorf("got %q", results[0].Name)
	}
}

func TestSkillSearch_MatchesByDescription(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "s1", Description: "analyze performance"})
	r.Register(&Skill{Name: "s2", Description: "write tests"})

	results := r.Search("performance")
	if len(results) != 1 || results[0].Name != "s1" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestSkillSearch_EmptyReturnsAll(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "a"})
	r.Register(&Skill{Name: "b"})

	results := r.Search("")
	if len(results) != 2 {
		t.Errorf("empty search should return all, got %d", len(results))
	}
}

func TestSkillSearch_MultiTermAND(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "git-commit", Description: "commit code changes"})
	r.Register(&Skill{Name: "git-log", Description: "view git history"})

	results := r.Search("git commit")
	if len(results) != 1 || results[0].Name != "git-commit" {
		t.Errorf("multi-term AND failed: got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Bundled skills
// ---------------------------------------------------------------------------

func TestLoadBundledSkills_NonEmpty(t *testing.T) {
	bundled, err := LoadBundledSkills()
	if err != nil {
		t.Fatalf("LoadBundledSkills: %v", err)
	}
	if len(bundled) == 0 {
		t.Fatal("expected at least one bundled skill")
	}
	for _, s := range bundled {
		if s.Source != "bundled" {
			t.Errorf("skill %q source = %q, want 'bundled'", s.Name, s.Source)
		}
		if s.Content == "" {
			t.Errorf("skill %q has empty content", s.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// LoadSkillsFromDir
// ---------------------------------------------------------------------------

func TestLoadSkillsFromDir_NonexistentReturnsNil(t *testing.T) {
	skills, err := LoadSkillsFromDir("/nonexistent/path", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skills != nil {
		t.Error("expected nil for nonexistent dir")
	}
}

func TestLoadSkillsFromDir_ValidDir(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\nname: my-skill\ndescription: test\nuser_invocable: true\n---\nDo the thing."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadSkillsFromDir(dir, "project")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "my-skill" {
		t.Errorf("name = %q", skills[0].Name)
	}
	if skills[0].Source != "project" {
		t.Errorf("source = %q", skills[0].Source)
	}
}

// ---------------------------------------------------------------------------
// SkillTool
// ---------------------------------------------------------------------------

func TestSkillTool_CallKnownSkill(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{
		Name:        "greet",
		Description: "say hello",
		Content:     "Hello, world!",
	})

	tool := NewSkillTool(r)
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"skill_name": "greet",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	data, ok := result.Data.(string)
	if !ok {
		t.Fatalf("result.Data is not string: %T", result.Data)
	}
	if !strings.Contains(data, "Hello, world!") {
		t.Errorf("result should contain skill content, got %q", data)
	}
}

func TestSkillTool_CallWithArgs(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(&Skill{Name: "gen", Content: "Generate code"})

	tool := NewSkillTool(r)
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"skill_name": "gen",
		"args":       "for a REST API",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	data := result.Data.(string)
	if !strings.Contains(data, "User Context") || !strings.Contains(data, "REST API") {
		t.Errorf("result should contain user context, got %q", data)
	}
}

func TestSkillTool_CallUnknownSkill(t *testing.T) {
	r := NewSkillRegistry()
	tool := NewSkillTool(r)
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"skill_name": "nonexistent",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	data := result.Data.(string)
	if !strings.Contains(data, "Unknown skill") {
		t.Errorf("expected 'Unknown skill' message, got %q", data)
	}
}

func TestSkillTool_MissingRequired(t *testing.T) {
	r := NewSkillRegistry()
	tool := NewSkillTool(r)
	_, err := tool.Call(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing skill_name")
	}
}

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------

func TestManager_ExecuteSkill(t *testing.T) {
	mgr := NewManager("")
	mgr.Registry().Register(&Skill{Name: "hello", Content: "Hello!"})

	content, err := mgr.ExecuteSkill("hello", "")
	if err != nil {
		t.Fatalf("ExecuteSkill error: %v", err)
	}
	if content != "Hello!" {
		t.Errorf("content = %q", content)
	}
}

func TestManager_ExecuteSkillWithArgs(t *testing.T) {
	mgr := NewManager("")
	mgr.Registry().Register(&Skill{Name: "hello", Content: "Hello!"})

	content, err := mgr.ExecuteSkill("hello", "extra context")
	if err != nil {
		t.Fatalf("ExecuteSkill error: %v", err)
	}
	if !strings.Contains(content, "Hello!") || !strings.Contains(content, "extra context") {
		t.Errorf("content = %q", content)
	}
}

func TestManager_ExecuteSkillNotFound(t *testing.T) {
	mgr := NewManager("")
	_, err := mgr.ExecuteSkill("nonexistent", "")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}
