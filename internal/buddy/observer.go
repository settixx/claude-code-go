package buddy

import (
	"sync"
	"time"
)

// BuddyEvent represents an event the companion can react to.
type BuddyEvent string

const (
	EventUserMessage       BuddyEvent = "user_message"
	EventAssistantResponse BuddyEvent = "assistant_response"
	EventToolSuccess       BuddyEvent = "tool_success"
	EventToolError         BuddyEvent = "tool_error"
	EventSessionStart      BuddyEvent = "session_start"
	EventSessionEnd        BuddyEvent = "session_end"
	EventIdle              BuddyEvent = "idle"
)

const defaultIdleTimeout = 30 * time.Second

// Observer watches events and drives companion reactions.
type Observer struct {
	companion   *Companion
	lastEvent   time.Time
	idleTimeout time.Duration
	reaction    string

	mu       sync.Mutex
	stopIdle chan struct{}
	running  bool
}

// NewObserver creates an observer bound to the given companion.
func NewObserver(c *Companion) *Observer {
	return &Observer{
		companion:   c,
		idleTimeout: defaultIdleTimeout,
		stopIdle:    make(chan struct{}),
	}
}

// Observe processes an event, updates companion mood, and stores the reaction.
func (o *Observer) Observe(event BuddyEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.lastEvent = time.Now()
	o.reaction = o.companion.React(string(event))
	o.resetIdleTimerLocked()
}

// LastReaction returns the most recent reaction text.
func (o *Observer) LastReaction() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.reaction
}

// StartIdleTimer begins a background goroutine that puts the companion
// to sleep after idleTimeout seconds of inactivity.
func (o *Observer) StartIdleTimer() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return
	}
	o.running = true
	o.lastEvent = time.Now()
	go o.idleLoop()
}

// StopIdleTimer stops the background idle watcher.
func (o *Observer) StopIdleTimer() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return
	}
	o.running = false
	close(o.stopIdle)
}

func (o *Observer) idleLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopIdle:
			return
		case <-ticker.C:
			o.mu.Lock()
			elapsed := time.Since(o.lastEvent)
			if elapsed >= o.idleTimeout && o.companion.Mood != MoodSleeping {
				o.reaction = o.companion.React(string(EventIdle))
			}
			o.mu.Unlock()
		}
	}
}

func (o *Observer) resetIdleTimerLocked() {
	if !o.running {
		return
	}
	// Signal the old goroutine to stop, start a fresh one.
	close(o.stopIdle)
	o.stopIdle = make(chan struct{})
	go o.idleLoop()
}
