package tasktool

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TaskEntry represents a single task tracked in the in-memory store.
type TaskEntry struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	AgentType   string    `json:"agent_type,omitempty"`
	Status      string    `json:"status"`
	Result      string    `json:"result,omitempty"`
	Output      string    `json:"output,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskStore is a thread-safe in-memory store for task entries.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*TaskEntry
}

// NewTaskStore creates an empty task store.
func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]*TaskEntry)}
}

// Create inserts a new task and returns it.
func (s *TaskStore) Create(desc, agentType string) *TaskEntry {
	id := generateTaskID()
	now := time.Now()
	e := &TaskEntry{
		ID:          id,
		Description: desc,
		AgentType:   agentType,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.mu.Lock()
	s.tasks[id] = e
	s.mu.Unlock()

	return e
}

// Get returns a snapshot of a task by ID.
func (s *TaskStore) Get(id string) (*TaskEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	cp := *e
	return &cp, true
}

// List returns all tasks, optionally filtered by status.
func (s *TaskStore) List(statusFilter string) []*TaskEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*TaskEntry, 0, len(s.tasks))
	for _, e := range s.tasks {
		if statusFilter != "" && e.Status != statusFilter {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out
}

// Update sets status and optional result on an existing task.
func (s *TaskStore) Update(id, status, result string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	e.Status = status
	if result != "" {
		e.Result = result
	}
	e.UpdatedAt = time.Now()
	return nil
}

// AppendOutput appends log/output text to a task.
func (s *TaskStore) AppendOutput(id, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	e.Output += text
	e.UpdatedAt = time.Now()
	return nil
}

// Stop sets a task's status to "stopped" if it is currently running or pending.
func (s *TaskStore) Stop(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	if e.Status != "running" && e.Status != "pending" {
		return fmt.Errorf("task %q cannot be stopped (status: %s)", id, e.Status)
	}
	e.Status = "stopped"
	e.UpdatedAt = time.Now()
	return nil
}

func generateTaskID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "task-" + hex.EncodeToString(b)
}
