package config

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/errors"
	"github.com/settixx/claude-code-go/internal/types"
)

// Provider loads, merges, and exposes configuration from env vars,
// user-level settings, project-level settings, and CLAUDE.md files.
// It implements interfaces.ConfigProvider.
type Provider struct {
	mu          sync.RWMutex
	cwd         string
	user        types.UserConfig
	project     types.ProjectConfig
	claudeRules *ClaudeMDRules
}

// NewProvider creates a Provider rooted at cwd, immediately loading all config.
func NewProvider(cwd string) (*Provider, error) {
	p := &Provider{cwd: cwd}
	if err := p.load(); err != nil {
		return nil, err
	}
	return p, nil
}

// GetSettings returns the fully resolved settings (user + project merged).
func (p *Provider) GetSettings() types.Settings {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return types.Settings{
		User:    p.user,
		Project: p.project,
	}
}

// GetProjectConfig returns the project-level configuration.
func (p *Provider) GetProjectConfig() types.ProjectConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.project
}

// GetUserConfig returns the user-level configuration.
func (p *Provider) GetUserConfig() types.UserConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.user
}

// GetClaudeMDRules returns the aggregated CLAUDE.md permission rules.
func (p *Provider) GetClaudeMDRules() *ClaudeMDRules {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.claudeRules
}

// Reload re-reads every config source and replaces the in-memory values.
func (p *Provider) Reload() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.load()
}

// load performs the actual I/O. Caller must hold p.mu (write).
// Load order: env vars → user config → project config → CLAUDE.md
func (p *Provider) load() error {
	p.applyEnvVars()

	if err := p.loadUserConfig(); err != nil {
		return err
	}
	if err := p.loadProjectConfig(); err != nil {
		return err
	}

	rules, _ := LoadClaudeMD(p.cwd)
	p.claudeRules = rules
	return nil
}

func (p *Provider) applyEnvVars() {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		p.user.CustomApiKey = key
	}
	if model := os.Getenv("CLAUDE_MODEL"); model != "" {
		p.user.DefaultModel = model
	}
}

func (p *Provider) loadUserConfig() error {
	path := UserSettingsPath()
	if err := loadJSONFile(path, &p.user); err != nil {
		return err
	}
	localPath := strings.TrimSuffix(path, ".json") + ".local.json"
	return loadJSONFile(localPath, &p.user)
}

func (p *Provider) loadProjectConfig() error {
	path := ProjectSettingsPath(p.cwd)
	if err := loadJSONFile(path, &p.project); err != nil {
		return err
	}
	localPath := strings.TrimSuffix(path, ".json") + ".local.json"
	return loadJSONFile(localPath, &p.project)
}

// loadJSONFile reads a JSON file into dst. Missing files are silently
// ignored; malformed files produce a ConfigParseError.
func loadJSONFile(path string, dst interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return &errors.ConfigParseError{
			FilePath: path,
			Cause:    err,
		}
	}
	return nil
}
