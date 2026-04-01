package storage

import "testing"

func TestSaveMemoryAndLoadMemoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	content := "# Project Memory\n\n- Use Go 1.23\n- Run tests with `go test ./...`\n"

	if err := SaveMemory(dir, content); err != nil {
		t.Fatalf("SaveMemory error: %v", err)
	}

	loaded, err := LoadMemory(dir)
	if err != nil {
		t.Fatalf("LoadMemory error: %v", err)
	}

	if loaded != content {
		t.Errorf("LoadMemory() = %q, want %q", loaded, content)
	}
}

func TestLoadMemoryNonexistent(t *testing.T) {
	dir := t.TempDir()

	loaded, err := LoadMemory(dir)
	if err != nil {
		t.Fatalf("LoadMemory error: %v", err)
	}
	if loaded != "" {
		t.Errorf("LoadMemory on empty dir should return empty string, got %q", loaded)
	}
}

func TestSaveMemoryOverwrites(t *testing.T) {
	dir := t.TempDir()

	if err := SaveMemory(dir, "version 1"); err != nil {
		t.Fatalf("SaveMemory error: %v", err)
	}
	if err := SaveMemory(dir, "version 2"); err != nil {
		t.Fatalf("SaveMemory error: %v", err)
	}

	loaded, err := LoadMemory(dir)
	if err != nil {
		t.Fatalf("LoadMemory error: %v", err)
	}
	if loaded != "version 2" {
		t.Errorf("LoadMemory() = %q, want %q", loaded, "version 2")
	}
}
