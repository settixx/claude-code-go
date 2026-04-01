package coordinator

import (
	"fmt"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// Team groups a set of worker agents under a named leader.
type Team struct {
	Name    string
	Leader  types.AgentId
	Members []types.AgentId
}

// TeamManager tracks named teams and provides CRUD operations.
type TeamManager struct {
	mu    sync.RWMutex
	teams map[string]*Team
	pool  *WorkerPool
}

// NewTeamManager creates a manager that uses the given pool for worker lookups.
func NewTeamManager(pool *WorkerPool) *TeamManager {
	return &TeamManager{
		teams: make(map[string]*Team),
		pool:  pool,
	}
}

// Create registers a new team. The first member is assigned as leader by default.
func (tm *TeamManager) Create(name string, members []types.AgentId) (*Team, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.teams[name]; exists {
		return nil, fmt.Errorf("team %q already exists", name)
	}
	if len(members) == 0 {
		return nil, fmt.Errorf("team must have at least one member")
	}

	t := &Team{
		Name:    name,
		Leader:  members[0],
		Members: members,
	}
	tm.teams[name] = t
	return t, nil
}

// Delete removes a team entry. It does NOT stop the member workers.
func (tm *TeamManager) Delete(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, exists := tm.teams[name]; !exists {
		return fmt.Errorf("team %q not found", name)
	}
	delete(tm.teams, name)
	return nil
}

// Get returns a team by name.
func (tm *TeamManager) Get(name string) (*Team, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	t, ok := tm.teams[name]
	return t, ok
}

// List returns the names of all registered teams.
func (tm *TeamManager) List() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	names := make([]string, 0, len(tm.teams))
	for n := range tm.teams {
		names = append(names, n)
	}
	return names
}

// ShutdownTeam gracefully stops every member in the named team and then
// deletes the team entry.
func (tm *TeamManager) ShutdownTeam(name string) error {
	tm.mu.Lock()
	t, ok := tm.teams[name]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("team %q not found", name)
	}
	members := make([]types.AgentId, len(t.Members))
	copy(members, t.Members)
	delete(tm.teams, name)
	tm.mu.Unlock()

	var firstErr error
	for _, id := range members {
		if err := tm.pool.StopWorker(id); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
