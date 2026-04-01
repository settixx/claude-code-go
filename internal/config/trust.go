package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TrustedProjects holds the list of project directories the user has trusted.
type TrustedProjects struct {
	Paths []string `json:"trusted_paths"`
}

// TrustFilePath returns the path to the trust database.
func TrustFilePath() string {
	return filepath.Join(ConfigDir(), "trusted_projects.json")
}

// IsProjectTrusted checks if the given directory is in the trust list.
func IsProjectTrusted(dir string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	tp := loadTrustedProjects()
	for _, p := range tp.Paths {
		if p == absDir {
			return true
		}
	}
	return false
}

// TrustProject adds a directory to the trust list.
func TrustProject(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	tp := loadTrustedProjects()
	for _, p := range tp.Paths {
		if p == absDir {
			return nil
		}
	}

	tp.Paths = append(tp.Paths, absDir)
	return saveTrustedProjects(tp)
}

func loadTrustedProjects() *TrustedProjects {
	tp := &TrustedProjects{}
	data, err := os.ReadFile(TrustFilePath())
	if err != nil {
		return tp
	}
	json.Unmarshal(data, tp)
	return tp
}

func saveTrustedProjects(tp *TrustedProjects) error {
	dir := filepath.Dir(TrustFilePath())
	os.MkdirAll(dir, 0o755)
	data, err := json.MarshalIndent(tp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TrustFilePath(), data, 0o644)
}
