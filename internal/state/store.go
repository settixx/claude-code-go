package state

import (
	"sync"
	"sync/atomic"

	"github.com/settixx/claude-code-go/internal/types"
)

// Store is a thread-safe, reactive container for AppState.
// It implements interfaces.StateStore.
type Store struct {
	mu          sync.RWMutex
	state       types.AppState
	subscribers map[uint64]func(types.AppState)
	nextID      atomic.Uint64
}

// NewStore creates a Store initialised with the given state snapshot.
func NewStore(initial types.AppState) *Store {
	return &Store{
		state:       initial,
		subscribers: make(map[uint64]func(types.AppState)),
	}
}

// Get returns a snapshot of the current state.
// The returned value is a copy; callers may read it without holding any lock.
func (s *Store) Get() types.AppState {
	s.mu.RLock()
	snap := s.state
	s.mu.RUnlock()
	return snap
}

// Update applies fn atomically and notifies all subscribers with the new state.
func (s *Store) Update(fn func(*types.AppState)) {
	s.mu.Lock()
	fn(&s.state)
	snap := s.state
	subs := s.cloneSubscribers()
	s.mu.Unlock()

	for _, cb := range subs {
		cb(snap)
	}
}

// Subscribe registers a callback that fires after every Update.
// It returns an unsubscribe function; calling it removes the callback.
func (s *Store) Subscribe(fn func(types.AppState)) func() {
	id := s.nextID.Add(1)

	s.mu.Lock()
	s.subscribers[id] = fn
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		delete(s.subscribers, id)
		s.mu.Unlock()
	}
}

// cloneSubscribers returns a snapshot of the subscriber map so callbacks
// can be invoked outside the critical section.
func (s *Store) cloneSubscribers() []func(types.AppState) {
	out := make([]func(types.AppState), 0, len(s.subscribers))
	for _, fn := range s.subscribers {
		out = append(out, fn)
	}
	return out
}
