package coordinator

import (
	"fmt"
	"sync"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// AgentTask wraps a TaskState with a reference to the worker that owns it,
// plus progress tracking and a small state machine:
//
//	pending → running → complete | failed | stopped
type AgentTask struct {
	mu       sync.Mutex
	state    types.TaskState
	worker   *Worker
	progress []ProgressEntry
}

// ProgressEntry records a timestamped progress notification.
type ProgressEntry struct {
	Timestamp time.Time
	Message   string
}

// NewAgentTask creates a task in pending status linked to the given worker.
func NewAgentTask(id string, kind types.TaskKind, name string, worker *Worker) *AgentTask {
	return &AgentTask{
		state: types.TaskState{
			ID:     id,
			Kind:   kind,
			Status: types.TaskPending,
			Name:   name,
			AgentID: worker.ID,
		},
		worker: worker,
	}
}

// State returns a snapshot of the underlying TaskState.
func (t *AgentTask) State() types.TaskState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// Worker returns the associated worker.
func (t *AgentTask) Worker() *Worker {
	return t.worker
}

// Start transitions pending → running.
func (t *AgentTask) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state.Status != types.TaskPending {
		return fmt.Errorf("cannot start task %s: status is %s", t.state.ID, t.state.Status)
	}
	t.state.Status = types.TaskRunning
	return nil
}

// Complete transitions running → complete.
func (t *AgentTask) Complete() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state.Status != types.TaskRunning {
		return fmt.Errorf("cannot complete task %s: status is %s", t.state.ID, t.state.Status)
	}
	t.state.Status = types.TaskComplete
	return nil
}

// Fail transitions running → failed.
func (t *AgentTask) Fail() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state.Status != types.TaskRunning {
		return fmt.Errorf("cannot fail task %s: status is %s", t.state.ID, t.state.Status)
	}
	t.state.Status = types.TaskFailed
	return nil
}

// Stop transitions running → stopped.
func (t *AgentTask) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state.Status != types.TaskRunning {
		return fmt.Errorf("cannot stop task %s: status is %s", t.state.ID, t.state.Status)
	}
	t.state.Status = types.TaskStopped
	return nil
}

// NotifyProgress appends a timestamped progress message.
func (t *AgentTask) NotifyProgress(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progress = append(t.progress, ProgressEntry{
		Timestamp: time.Now(),
		Message:   msg,
	})
}

// Progress returns a copy of all recorded progress entries.
func (t *AgentTask) Progress() []ProgressEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]ProgressEntry, len(t.progress))
	copy(cp, t.progress)
	return cp
}

// SetWorktreePath assigns the worktree path on the underlying TaskState.
func (t *AgentTask) SetWorktreePath(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.WorktreePath = path
}

// SetBackgrounded marks the task as backgrounded.
func (t *AgentTask) SetBackgrounded(v bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.IsBackgrounded = v
}
