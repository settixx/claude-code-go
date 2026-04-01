package tui

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner renders a text-based progress spinner on a single terminal line.
type Spinner struct {
	mu      sync.Mutex
	text    string
	running bool
	stop    chan struct{}
	done    chan struct{}
}

// NewSpinner creates a spinner with the given label text.
func NewSpinner(text string) *Spinner {
	return &Spinner{
		text: text,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

// Start begins the spinner animation in a background goroutine.
// Calling Start on an already-running spinner is a no-op.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.loop()
}

// Stop halts the spinner animation and clears the spinner line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stop)
	s.mu.Unlock()

	<-s.done
	fmt.Fprintf(os.Stdout, "\r%s\r", AnsiClearLine)
}

// UpdateText changes the spinner label while it is running.
func (s *Spinner) UpdateText(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text = text
}

func (s *Spinner) loop() {
	defer close(s.done)

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	idx := 0
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			frame := Cyan(spinnerFrames[idx%len(spinnerFrames)])
			text := s.text
			s.mu.Unlock()

			fmt.Fprintf(os.Stdout, "\r%s %s %s", AnsiClearLine, frame, text)
			idx++
		}
	}
}
