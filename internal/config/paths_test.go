package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDirNonEmpty(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Error("ConfigDir() returned empty string")
	}
	if !strings.Contains(dir, "claude") {
		t.Errorf("ConfigDir() = %q, should contain 'claude'", dir)
	}
}

func TestDataDirNonEmpty(t *testing.T) {
	dir := DataDir()
	if dir == "" {
		t.Error("DataDir() returned empty string")
	}
	if !strings.Contains(dir, "claude") {
		t.Errorf("DataDir() = %q, should contain 'claude'", dir)
	}
}

func TestConfigDirRespectsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := ConfigDir()
	want := filepath.Join(tmp, "claude")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestDataDirRespectsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	dir := DataDir()
	want := filepath.Join(tmp, "claude")
	if dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestSessionDir(t *testing.T) {
	dir := SessionDir()
	if !strings.HasSuffix(dir, "sessions") {
		t.Errorf("SessionDir() = %q, should end with 'sessions'", dir)
	}
}

func TestMemoryDir(t *testing.T) {
	dir := MemoryDir("/home/user/project")
	want := filepath.Join("/home/user/project", ".claude")
	if dir != want {
		t.Errorf("MemoryDir() = %q, want %q", dir, want)
	}
}

func TestUserSettingsPath(t *testing.T) {
	path := UserSettingsPath()
	if !strings.HasSuffix(path, "settings.json") {
		t.Errorf("UserSettingsPath() = %q, should end with 'settings.json'", path)
	}
}

func TestProjectSettingsPath(t *testing.T) {
	path := ProjectSettingsPath("/tmp/proj")
	want := filepath.Join("/tmp/proj", ".claude", "settings.json")
	if path != want {
		t.Errorf("ProjectSettingsPath() = %q, want %q", path, want)
	}
}

func TestHomeDirFallback(t *testing.T) {
	original := os.Getenv("HOME")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	dir := homeDir()

	t.Setenv("HOME", original)

	if dir == "" {
		t.Error("homeDir() returned empty string with no HOME")
	}
}
