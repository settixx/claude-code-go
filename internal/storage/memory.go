package storage

import (
	"os"
	"path/filepath"
)

const memoryFileName = "memory.md"

// LoadMemory reads the .claude/memory.md file from dir.
// Returns an empty string (no error) when the file does not exist.
func LoadMemory(dir string) (string, error) {
	data, err := os.ReadFile(memoryPath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// SaveMemory writes content to .claude/memory.md inside dir,
// creating the .claude directory if needed.
func SaveMemory(dir string, content string) error {
	memDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(memoryPath(dir), []byte(content), 0o644)
}

func memoryPath(dir string) string {
	return filepath.Join(dir, ".claude", memoryFileName)
}
