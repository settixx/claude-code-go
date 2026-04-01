package coordinator

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/settixx/claude-code-go/internal/types"
)

// Team groups a set of worker agents under a named leader.
type Team struct {
	Name    string
	Leader  types.AgentId
	Members []types.AgentId

	pool *WorkerPool
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
		pool:    tm.pool,
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

// Start spawns all team members as LLM-backed agents via the orchestrator.
// Members must already be registered in the pool (typically by Orchestrator.SpawnTeam).
func (t *Team) Start(ctx context.Context, orch *Orchestrator) error {
	for _, id := range t.Members {
		if _, ok := orch.Pool.Get(id); !ok {
			return fmt.Errorf("team %q: member %s not found in pool", t.Name, id)
		}
	}
	return nil
}

// WaitAll blocks until every team member completes. Returns the first error.
func (t *Team) WaitAll() error {
	if t.pool == nil {
		return fmt.Errorf("team %q: no pool reference", t.Name)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, id := range t.Members {
		w, ok := t.pool.Get(id)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(worker *Worker) {
			defer wg.Done()
			<-worker.Done()
			worker.mu.Lock()
			err := worker.Err
			worker.mu.Unlock()
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
	return firstErr
}

// CollectResults aggregates the result text from all team members.
// Returns a map of worker-name → result and a formatted summary string.
func (t *Team) CollectResults() (map[string]string, string) {
	if t.pool == nil {
		return nil, ""
	}

	results := make(map[string]string, len(t.Members))
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("## Team %q Results\n\n", t.Name))

	for _, id := range t.Members {
		w, ok := t.pool.Get(id)
		if !ok {
			continue
		}
		w.mu.Lock()
		name := w.Name
		result := w.Result
		status := w.Status
		w.mu.Unlock()

		results[name] = result

		summary.WriteString(fmt.Sprintf("### %s [%s]\n", name, status))
		if result != "" {
			summary.WriteString(result)
		} else {
			summary.WriteString("(no output)")
		}
		summary.WriteString("\n\n")
	}

	return results, summary.String()
}
