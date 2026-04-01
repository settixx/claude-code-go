package config

import (
	"testing"
)

func TestParseFrontmatter_NonePresent(t *testing.T) {
	content := "# Just a heading\nSome body text."
	fm, remaining := ParseFrontmatter(content)
	if fm != nil {
		t.Error("expected nil frontmatter when none present")
	}
	if remaining != content {
		t.Errorf("remaining = %q, want original content", remaining)
	}
}

func TestParseFrontmatter_WithPaths(t *testing.T) {
	content := "---\n- /home/user/project/**\n- /tmp/*\n---\nBody content here."
	fm, remaining := ParseFrontmatter(content)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if len(fm.Paths) != 2 {
		t.Fatalf("Paths = %v, want 2 entries", fm.Paths)
	}
	if fm.Paths[0] != "/home/user/project/**" {
		t.Errorf("Paths[0] = %q", fm.Paths[0])
	}
	if fm.Paths[1] != "/tmp/*" {
		t.Errorf("Paths[1] = %q", fm.Paths[1])
	}
	if remaining != "\nBody content here." {
		t.Errorf("remaining = %q", remaining)
	}
}

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	// Empty frontmatter block: "---\n---\n" — the closing "---" immediately
	// follows the opening, so content[4:] starts with "---" and there's no
	// "\n---" to find (the closing delimiter needs a preceding newline).
	// This means ParseFrontmatter returns nil for truly empty blocks.
	content := "---\n---\nBody."
	fm, _ := ParseFrontmatter(content)
	if fm != nil {
		t.Error("truly empty frontmatter block returns nil (no content between delimiters)")
	}
}

func TestParseFrontmatter_WithContentBetweenDelimiters(t *testing.T) {
	content := "---\nsome_key: value\n---\nBody."
	fm, remaining := ParseFrontmatter(content)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter with content between delimiters")
	}
	// "some_key: value" doesn't start with "- " so no paths
	if len(fm.Paths) != 0 {
		t.Errorf("Paths = %v, want empty (no list items)", fm.Paths)
	}
	if remaining != "\nBody." {
		t.Errorf("remaining = %q", remaining)
	}
}

func TestParseFrontmatter_NoClosingDelimiter(t *testing.T) {
	content := "---\n- path\nno closing"
	fm, remaining := ParseFrontmatter(content)
	if fm != nil {
		t.Error("expected nil frontmatter when no closing delimiter")
	}
	if remaining != content {
		t.Error("remaining should be original content")
	}
}

func TestParseFrontmatter_QuotedPaths(t *testing.T) {
	content := "---\n- \"/some/path\"\n- '/other/path'\n---\nBody."
	fm, _ := ParseFrontmatter(content)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if len(fm.Paths) != 2 {
		t.Fatalf("Paths = %v, want 2", fm.Paths)
	}
	if fm.Paths[0] != "/some/path" {
		t.Errorf("Paths[0] = %q, quotes should be stripped", fm.Paths[0])
	}
}

func TestShouldApply_NilFrontmatter(t *testing.T) {
	var fm *Frontmatter
	if !fm.ShouldApply("/any/path") {
		t.Error("nil frontmatter should always apply")
	}
}

func TestShouldApply_EmptyPaths(t *testing.T) {
	fm := &Frontmatter{}
	if !fm.ShouldApply("/any/path") {
		t.Error("empty Paths should always apply")
	}
}

func TestShouldApply_ExactMatch(t *testing.T) {
	fm := &Frontmatter{Paths: []string{"/home/user/project"}}

	if !fm.ShouldApply("/home/user/project") {
		t.Error("exact match should apply")
	}
	if fm.ShouldApply("/home/user/other") {
		t.Error("non-matching path should not apply")
	}
}

func TestShouldApply_GlobstarPattern(t *testing.T) {
	fm := &Frontmatter{Paths: []string{"/home/user/**"}}

	if !fm.ShouldApply("/home/user/project/sub") {
		t.Error("globstar pattern should match nested paths")
	}
	if fm.ShouldApply("/other/path") {
		t.Error("globstar should not match unrelated paths")
	}
}

func TestShouldApply_MultiplePaths(t *testing.T) {
	fm := &Frontmatter{Paths: []string{"/a", "/b/**"}}

	if !fm.ShouldApply("/a") {
		t.Error("first path should match")
	}
	if !fm.ShouldApply("/b/sub/deep") {
		t.Error("second globstar path should match")
	}
	if fm.ShouldApply("/c") {
		t.Error("unmatched path should not apply")
	}
}

// ---------------------------------------------------------------------------
// Frontmatter — additional edge cases
// ---------------------------------------------------------------------------

func TestParseFrontmatter_MultipleListItems(t *testing.T) {
	content := "---\n- /path/a\n- /path/b\n- /path/c\n---\nBody."
	fm, remaining := ParseFrontmatter(content)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if len(fm.Paths) != 3 {
		t.Fatalf("Paths = %v, want 3 entries", fm.Paths)
	}
	if remaining != "\nBody." {
		t.Errorf("remaining = %q", remaining)
	}
}

func TestParseFrontmatter_MixedContent(t *testing.T) {
	content := "---\nname: test\n- /path/a\ndescription: something\n- /path/b\n---\nBody."
	fm, _ := ParseFrontmatter(content)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if len(fm.Paths) != 2 {
		t.Errorf("expected 2 paths from list items, got %d: %v", len(fm.Paths), fm.Paths)
	}
}

func TestShouldApply_SingleWildcard(t *testing.T) {
	fm := &Frontmatter{Paths: []string{"/tmp/*"}}

	if !fm.ShouldApply("/tmp/x") {
		t.Error("/tmp/x should match /tmp/*")
	}
	if fm.ShouldApply("/tmp/x/y") {
		t.Error("/tmp/x/y should not match single wildcard /tmp/*")
	}
}
