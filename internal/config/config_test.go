package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONFile_NonExistent(t *testing.T) {
	var dst struct{ Key string }
	err := loadJSONFile("/nonexistent/path.json", &dst)
	if err != nil {
		t.Errorf("nonexistent file should return nil error, got %v", err)
	}
}

func TestLoadJSONFile_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := map[string]string{"Key": "value"}
	raw, _ := json.Marshal(data)
	os.WriteFile(path, raw, 0o644)

	var dst struct{ Key string }
	err := loadJSONFile(path, &dst)
	if err != nil {
		t.Fatalf("loadJSONFile: %v", err)
	}
	if dst.Key != "value" {
		t.Errorf("Key = %q, want value", dst.Key)
	}
}

func TestLoadJSONFile_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("{invalid}"), 0o644)

	var dst struct{ Key string }
	err := loadJSONFile(path, &dst)
	if err == nil {
		t.Error("malformed JSON should return error")
	}
}

func TestNewProvider_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_MODEL", "")

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	settings := provider.GetSettings()
	_ = settings

	rules := provider.GetClaudeMDRules()
	if rules == nil {
		t.Error("ClaudeMDRules should not be nil even for empty dir")
	}
}

func TestNewProvider_WithEnvVars(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")
	t.Setenv("CLAUDE_MODEL", "claude-test")

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	userCfg := provider.GetUserConfig()
	if userCfg.CustomApiKey != "sk-test-key" {
		t.Errorf("CustomApiKey = %q, want sk-test-key", userCfg.CustomApiKey)
	}
	if userCfg.DefaultModel != "claude-test" {
		t.Errorf("DefaultModel = %q, want claude-test", userCfg.DefaultModel)
	}
}

func TestProvider_Reload(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_MODEL", "")

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	// Now create a CLAUDE.md and reload
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Allowed tools\n- FileRead\n"), 0o644)
	err = provider.Reload()
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rules := provider.GetClaudeMDRules()
	if len(rules.AllowPatterns) != 1 {
		t.Errorf("after reload, AllowPatterns = %v, want [FileRead]", rules.AllowPatterns)
	}
}

func TestProvider_LocalSettingsOverlay(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_MODEL", "")

	// Create user settings
	settingsDir := filepath.Join(tmp, "claude")
	os.MkdirAll(settingsDir, 0o755)
	os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{"defaultModel": "base-model"}`), 0o644)
	// Create local overlay that overrides
	os.WriteFile(filepath.Join(settingsDir, "settings.local.json"), []byte(`{"defaultModel": "local-model"}`), 0o644)

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	userCfg := provider.GetUserConfig()
	if userCfg.DefaultModel != "local-model" {
		t.Errorf("DefaultModel = %q, want local-model (local overlay should win)", userCfg.DefaultModel)
	}
}

// ---------------------------------------------------------------------------
// Provider — GetClaudeMDRules after CLAUDE.md exists
// ---------------------------------------------------------------------------

func TestProvider_GetClaudeMDRules_WithFile(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_MODEL", "")

	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Allowed tools\n- Bash\n- FileRead\n# Denied tools\n- WebFetch\n"), 0o644)

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	rules := provider.GetClaudeMDRules()
	if rules == nil {
		t.Fatal("rules should not be nil")
	}
	if len(rules.AllowPatterns) != 2 {
		t.Errorf("AllowPatterns = %v, want 2 entries", rules.AllowPatterns)
	}
	if len(rules.DenyPatterns) != 1 || rules.DenyPatterns[0] != "WebFetch" {
		t.Errorf("DenyPatterns = %v, want [WebFetch]", rules.DenyPatterns)
	}
}

// ---------------------------------------------------------------------------
// Provider — GetSettings returns merged data
// ---------------------------------------------------------------------------

func TestProvider_GetSettings(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("CLAUDE_MODEL", "test-model")

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	settings := provider.GetSettings()
	if settings.User.CustomApiKey != "test-key" {
		t.Errorf("User.CustomApiKey = %q, want test-key", settings.User.CustomApiKey)
	}
	if settings.User.DefaultModel != "test-model" {
		t.Errorf("User.DefaultModel = %q, want test-model", settings.User.DefaultModel)
	}
}

// ---------------------------------------------------------------------------
// Provider — GetProjectConfig from file
// ---------------------------------------------------------------------------

func TestProvider_GetProjectConfig(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_MODEL", "")

	projectCfgDir := filepath.Join(dir, ".claude")
	os.MkdirAll(projectCfgDir, 0o755)
	os.WriteFile(filepath.Join(projectCfgDir, "settings.json"), []byte(`{"projectName": "test-project"}`), 0o644)

	provider, err := NewProvider(dir)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	projCfg := provider.GetProjectConfig()
	_ = projCfg
}

// ---------------------------------------------------------------------------
// loadJSONFile — permissions error (unreadable file)
// ---------------------------------------------------------------------------

func TestLoadJSONFile_PermissionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noperm.json")
	os.WriteFile(path, []byte(`{"key":"val"}`), 0o000)

	var dst struct{ Key string }
	err := loadJSONFile(path, &dst)
	if err == nil {
		t.Error("expected error reading unreadable file")
	}

	// Restore perms for cleanup
	os.Chmod(path, 0o644)
}
