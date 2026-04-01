package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrustProject_AddAndCheck(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	projectDir := t.TempDir()

	if IsProjectTrusted(projectDir) {
		t.Error("project should not be trusted before adding")
	}

	err := TrustProject(projectDir)
	if err != nil {
		t.Fatalf("TrustProject: %v", err)
	}

	if !IsProjectTrusted(projectDir) {
		t.Error("project should be trusted after adding")
	}
}

func TestTrustProject_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	projectDir := t.TempDir()

	TrustProject(projectDir)
	TrustProject(projectDir) // second call should not duplicate

	tp := loadTrustedProjects()
	count := 0
	abs, _ := filepath.Abs(projectDir)
	for _, p := range tp.Paths {
		if p == abs {
			count++
		}
	}
	if count != 1 {
		t.Errorf("project should appear exactly once, got %d", count)
	}
}

func TestIsProjectTrusted_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if IsProjectTrusted("/some/random/dir") {
		t.Error("should not be trusted when trust file does not exist")
	}
}

func TestTrustFilePath_ContainsClaude(t *testing.T) {
	path := TrustFilePath()
	if path == "" {
		t.Error("TrustFilePath returned empty")
	}
}

func TestLoadTrustedProjects_MalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	trustFile := filepath.Join(tmp, "claude", "trusted_projects.json")
	os.MkdirAll(filepath.Dir(trustFile), 0o755)
	os.WriteFile(trustFile, []byte("not json"), 0o644)

	tp := loadTrustedProjects()
	if len(tp.Paths) != 0 {
		t.Errorf("malformed JSON should result in empty paths, got %v", tp.Paths)
	}
}

func TestSaveTrustedProjects(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	tp := &TrustedProjects{Paths: []string{"/a", "/b"}}
	err := saveTrustedProjects(tp)
	if err != nil {
		t.Fatalf("saveTrustedProjects: %v", err)
	}

	loaded := loadTrustedProjects()
	if len(loaded.Paths) != 2 {
		t.Errorf("loaded paths = %v, want 2 entries", loaded.Paths)
	}
}

// ---------------------------------------------------------------------------
// Trust — multiple projects
// ---------------------------------------------------------------------------

func TestTrustProject_MultipleProjects(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	TrustProject(dir1)
	TrustProject(dir2)

	if !IsProjectTrusted(dir1) {
		t.Error("dir1 should be trusted")
	}
	if !IsProjectTrusted(dir2) {
		t.Error("dir2 should be trusted")
	}

	tp := loadTrustedProjects()
	if len(tp.Paths) != 2 {
		t.Errorf("expected 2 trusted paths, got %d", len(tp.Paths))
	}
}

// ---------------------------------------------------------------------------
// Trust — relative vs absolute path
// ---------------------------------------------------------------------------

func TestIsProjectTrusted_ResolvesAbsolute(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := t.TempDir()
	TrustProject(dir)

	abs, _ := filepath.Abs(dir)
	if !IsProjectTrusted(abs) {
		t.Error("trusted project should be found by absolute path")
	}
}
