package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// Mailbox provides a filesystem-based message queue. Each agent gets a
// subdirectory under the root dir, and messages are stored as individual
// JSON files sorted by timestamp.
type Mailbox struct {
	dir string
	mu  sync.Mutex
}

// NewMailbox creates a Mailbox rooted at dir. The directory is created
// lazily on the first Send call.
func NewMailbox(dir string) *Mailbox {
	return &Mailbox{dir: dir}
}

// Send writes a message as a JSON file into the target agent's mailbox
// directory. The filename encodes a nanosecond timestamp for ordering.
func (m *Mailbox) Send(to types.AgentId, msg types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentDir := filepath.Join(m.dir, string(to))
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("create mailbox dir: %w", err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	name := fmt.Sprintf("%d.json", time.Now().UnixNano())
	path := filepath.Join(agentDir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write message file: %w", err)
	}
	return nil
}

// Receive reads and removes all pending messages for the given agent,
// returning them in chronological order.
func (m *Mailbox) Receive(id types.AgentId) ([]types.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentDir := filepath.Join(m.dir, string(id))
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mailbox dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return extractTimestamp(entries[i].Name()) < extractTimestamp(entries[j].Name())
	})

	var msgs []types.Message
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(agentDir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var msg types.Message
		if json.Unmarshal(data, &msg) == nil {
			msgs = append(msgs, msg)
		}
		_ = os.Remove(path)
	}
	return msgs, nil
}

// Poll returns a channel that yields messages as they arrive in the
// agent's mailbox. It checks the filesystem at the given interval.
// Close the context to stop polling.
func (m *Mailbox) Poll(ctx context.Context, id types.AgentId, interval time.Duration) <-chan types.Message {
	ch := make(chan types.Message, 32)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msgs, err := m.Receive(id)
				if err != nil {
					continue
				}
				for _, msg := range msgs {
					select {
					case ch <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch
}

func extractTimestamp(filename string) int64 {
	name := strings.TrimSuffix(filename, ".json")
	ts, _ := strconv.ParseInt(name, 10, 64)
	return ts
}
