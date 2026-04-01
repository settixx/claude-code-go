package bridge

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// SessionStatus describes the lifecycle state of a bridge session.
type SessionStatus string

const (
	StatusConnecting    SessionStatus = "connecting"
	StatusConnected     SessionStatus = "connected"
	StatusReconnecting  SessionStatus = "reconnecting"
	StatusDisconnected  SessionStatus = "disconnected"
)

// Session represents a single bridge session to a remote controller.
type Session struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Status    SessionStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`

	mu     sync.RWMutex
	bridge *Bridge
}

// SetStatus atomically updates the session status.
func (s *Session) SetStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	s.UpdatedAt = time.Now()
}

// GetStatus returns the current session status.
func (s *Session) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// Bridge returns the underlying bridge connection (may be nil).
func (s *Session) GetBridge() *Bridge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bridge
}

// SetBridge attaches a bridge connection to this session.
func (s *Session) SetBridge(b *Bridge) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bridge = b
}

// SessionStore manages the set of known bridge sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates an empty session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

// CreateSession initialises a new session with a random ID.
func (ss *SessionStore) CreateSession(name string) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("bridge: generate session id: %w", err)
	}

	now := time.Now()
	s := &Session{
		ID:        id,
		Name:      name,
		Status:    StatusDisconnected,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ss.mu.Lock()
	ss.sessions[id] = s
	ss.mu.Unlock()

	return s, nil
}

// ResumeSession looks up an existing session by ID.
func (ss *SessionStore) ResumeSession(id string) (*Session, error) {
	ss.mu.RLock()
	s, ok := ss.sessions[id]
	ss.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("bridge: session %q not found", id)
	}
	return s, nil
}

// List returns all tracked sessions (newest first).
func (ss *SessionStore) List() []*Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	out := make([]*Session, 0, len(ss.sessions))
	for _, s := range ss.sessions {
		out = append(out, s)
	}
	return out
}

// Remove deletes a session from the store.
func (ss *SessionStore) Remove(id string) {
	ss.mu.Lock()
	delete(ss.sessions, id)
	ss.mu.Unlock()
}

func generateSessionID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("bs-%x", b), nil
}
