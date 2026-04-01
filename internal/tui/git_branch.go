package tui

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

// GitBranchCache caches the current git branch name, refreshing periodically.
type GitBranchCache struct {
	mu       sync.Mutex
	branch   string
	fetchedAt time.Time
	ttl      time.Duration
}

// NewGitBranchCache returns a cache with the given refresh interval.
func NewGitBranchCache(ttl time.Duration) *GitBranchCache {
	return &GitBranchCache{ttl: ttl}
}

// Branch returns the cached branch name, refreshing if stale.
func (g *GitBranchCache) Branch() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Since(g.fetchedAt) < g.ttl && g.branch != "" {
		return g.branch
	}

	g.branch = fetchGitBranch()
	g.fetchedAt = time.Now()
	return g.branch
}

// Invalidate forces the next call to Branch() to re-fetch.
func (g *GitBranchCache) Invalidate() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fetchedAt = time.Time{}
}

func fetchGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
