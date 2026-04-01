package coordinator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// WorkerStatus describes the lifecycle state of a worker agent.
type WorkerStatus string

const (
	WorkerIdle     WorkerStatus = "idle"
	WorkerRunning  WorkerStatus = "running"
	WorkerStopped  WorkerStatus = "stopped"
	WorkerFailed   WorkerStatus = "failed"
	WorkerComplete WorkerStatus = "complete"
)

// Worker represents a single agent goroutine that processes messages
// independently inside an optional git worktree.
type Worker struct {
	ID           types.AgentId
	Name         string
	Status       WorkerStatus
	Prompt       string
	WorktreePath string

	mu       sync.Mutex
	messages []types.Message
	inbox    chan types.Message
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewWorker creates a Worker in idle state. Call Start to launch the goroutine.
func NewWorker(id types.AgentId, name string, prompt string) *Worker {
	return &Worker{
		ID:     id,
		Name:   name,
		Status: WorkerIdle,
		Prompt: prompt,
		inbox:  make(chan types.Message, 64),
		done:   make(chan struct{}),
	}
}

// Start launches the worker goroutine. It blocks in a message-processing loop
// until the context is cancelled or Stop is called.
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.Status == WorkerRunning {
		w.mu.Unlock()
		return fmt.Errorf("worker %s already running", w.ID)
	}
	ctx, w.cancel = context.WithCancel(ctx)
	w.Status = WorkerRunning
	w.mu.Unlock()

	go w.run(ctx)
	return nil
}

// Stop cancels the worker's context and waits for the goroutine to exit.
func (w *Worker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	<-w.done
}

// Send delivers a message to the worker's inbox channel.
func (w *Worker) Send(msg types.Message) {
	select {
	case w.inbox <- msg:
	default:
	}
}

// Messages returns a snapshot of all messages processed by this worker.
func (w *Worker) Messages() []types.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]types.Message, len(w.messages))
	copy(cp, w.messages)
	return cp
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.done)
	defer func() {
		w.mu.Lock()
		if w.Status == WorkerRunning {
			w.Status = WorkerStopped
		}
		w.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-w.inbox:
			if !ok {
				return
			}
			w.mu.Lock()
			w.messages = append(w.messages, msg)
			w.mu.Unlock()
			// The actual LLM query loop will be wired in by the caller
			// via a handler func; for now we record messages.
		}
	}
}

// WorkerPool manages the set of all active workers.
type WorkerPool struct {
	mu      sync.RWMutex
	workers map[types.AgentId]*Worker
	byName  map[string]types.AgentId
}

// NewWorkerPool creates an empty pool.
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: make(map[types.AgentId]*Worker),
		byName:  make(map[string]types.AgentId),
	}
}

// SpawnWorker creates a new worker, registers it, and starts its goroutine.
func (p *WorkerPool) SpawnWorker(ctx context.Context, name, prompt string) (*Worker, error) {
	id, err := generateAgentID(name)
	if err != nil {
		return nil, fmt.Errorf("generate agent id: %w", err)
	}

	w := NewWorker(id, name, prompt)

	p.mu.Lock()
	p.workers[id] = w
	p.byName[name] = id
	p.mu.Unlock()

	if err := w.Start(ctx); err != nil {
		return nil, err
	}
	return w, nil
}

// StopWorker gracefully stops a worker and removes it from the pool.
func (p *WorkerPool) StopWorker(id types.AgentId) error {
	p.mu.Lock()
	w, ok := p.workers[id]
	if !ok {
		p.mu.Unlock()
		return fmt.Errorf("worker %s not found", id)
	}
	delete(p.workers, id)
	delete(p.byName, w.Name)
	p.mu.Unlock()

	w.Stop()
	return nil
}

// Get returns a worker by ID.
func (p *WorkerPool) Get(id types.AgentId) (*Worker, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	w, ok := p.workers[id]
	return w, ok
}

// GetByName resolves a worker by its human-readable name.
func (p *WorkerPool) GetByName(name string) (*Worker, bool) {
	p.mu.RLock()
	id, ok := p.byName[name]
	if !ok {
		p.mu.RUnlock()
		return nil, false
	}
	w := p.workers[id]
	p.mu.RUnlock()
	return w, w != nil
}

// All returns a snapshot of every worker in the pool.
func (p *WorkerPool) All() []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Worker, 0, len(p.workers))
	for _, w := range p.workers {
		out = append(out, w)
	}
	return out
}

// BroadcastMessage sends a message to every worker's inbox.
func (p *WorkerPool) BroadcastMessage(msg types.Message) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, w := range p.workers {
		w.Send(msg)
	}
}

// StopAll gracefully stops all workers. Used during shutdown.
func (p *WorkerPool) StopAll() {
	p.mu.Lock()
	ids := make([]types.AgentId, 0, len(p.workers))
	for id := range p.workers {
		ids = append(ids, id)
	}
	p.mu.Unlock()

	for _, id := range ids {
		_ = p.StopWorker(id)
	}
}

func generateAgentID(name string) (types.AgentId, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(b)
	if name != "" {
		return types.AgentId(fmt.Sprintf("a-%s-%s", sanitizeName(name), suffix)), nil
	}
	return types.AgentId("a-" + suffix), nil
}

func sanitizeName(s string) string {
	out := make([]byte, 0, len(s))
	for i := range len(s) {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			out = append(out, c)
		} else if c >= 'A' && c <= 'Z' {
			out = append(out, c+32) // lowercase
		}
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return string(out)
}
