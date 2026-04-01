package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the user-level configuration directory.
// Respects XDG_CONFIG_HOME on Linux/macOS; falls back to ~/.claude.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "claude")
	}
	return filepath.Join(homeDir(), ".claude")
}

// DataDir returns the user-level data directory.
// Respects XDG_DATA_HOME on Linux/macOS; falls back to ~/.claude.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "claude")
	}
	return filepath.Join(homeDir(), ".claude")
}

// SessionDir returns the directory where session transcript files are stored.
func SessionDir() string {
	return filepath.Join(DataDir(), "sessions")
}

// MemoryDir returns the project-local .claude directory used for memory.md
// and per-project metadata. The caller supplies the project root.
func MemoryDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude")
}

// UserSettingsPath returns the full path to the user-global settings.json.
func UserSettingsPath() string {
	return filepath.Join(ConfigDir(), "settings.json")
}

// ProjectSettingsPath returns the full path to a project-level settings.json
// inside the given project root.
func ProjectSettingsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "settings.json")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return "/"
}
