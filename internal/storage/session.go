package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/types"
)

// FileStorage persists session transcripts as JSON files on disk.
// It implements interfaces.SessionStorage.
type FileStorage struct {
	dir string
}

// NewFileStorage creates a FileStorage rooted at dir.
// The directory is created on first Save if it does not exist.
func NewFileStorage(dir string) *FileStorage {
	return &FileStorage{dir: dir}
}

// Save writes the message slice as a JSON array to {dir}/{sessionID}.json.
func (fs *FileStorage) Save(sessionID types.SessionId, messages []types.Message) error {
	if err := os.MkdirAll(fs.dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fs.sessionPath(sessionID), data, 0o644)
}

// Load reads and parses a session transcript from disk.
func (fs *FileStorage) Load(sessionID types.SessionId) ([]types.Message, error) {
	data, err := os.ReadFile(fs.sessionPath(sessionID))
	if err != nil {
		return nil, err
	}

	var msgs []types.Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// List scans the session directory and returns summaries sorted by modification
// time descending (most recent first).
func (fs *FileStorage) List() ([]interfaces.SessionInfo, error) {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	infos := make([]interfaces.SessionInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := buildSessionInfo(fs.dir, e)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].UpdatedAt > infos[j].UpdatedAt
	})
	return infos, nil
}

// Delete removes the session file from disk.
func (fs *FileStorage) Delete(sessionID types.SessionId) error {
	return os.Remove(fs.sessionPath(sessionID))
}

func (fs *FileStorage) sessionPath(id types.SessionId) string {
	return filepath.Join(fs.dir, string(id)+".json")
}

func buildSessionInfo(dir string, entry os.DirEntry) (interfaces.SessionInfo, error) {
	name := entry.Name()
	id := types.SessionId(strings.TrimSuffix(name, ".json"))

	fi, err := entry.Info()
	if err != nil {
		return interfaces.SessionInfo{}, err
	}

	msgCount := estimateMessageCount(filepath.Join(dir, name))

	title := extractTitle(filepath.Join(dir, name))

	return interfaces.SessionInfo{
		ID:           id,
		Title:        title,
		UpdatedAt:    fi.ModTime().Unix(),
		MessageCount: msgCount,
	}, nil
}

// estimateMessageCount quickly counts top-level array entries by scanning
// for the "uuid" key without fully parsing the file.
func estimateMessageCount(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for i := 0; i < len(data)-5; i++ {
		if string(data[i:i+6]) == `"uuid"` {
			count++
		}
	}
	return count
}

// extractTitle reads the first user message text from the session file.
func extractTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var msgs []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &msgs); err != nil {
		return ""
	}

	for _, m := range msgs {
		if m.Type != "user" || m.Text == "" {
			continue
		}
		title := m.Text
		if len(title) > 80 {
			title = title[:80] + "…"
		}
		return title
	}
	return ""
}
