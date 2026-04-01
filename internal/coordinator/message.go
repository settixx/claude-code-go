package coordinator

import (
	"fmt"
	"strings"

	"github.com/settixx/claude-code-go/internal/types"
)

// RouteKind classifies how a message should be delivered.
type RouteKind string

const (
	RouteInProcess RouteKind = "in_process"
	RouteMailbox   RouteKind = "mailbox"
	RouteUDS       RouteKind = "uds"
	RouteBridge    RouteKind = "bridge"
)

// Router dispatches messages between the coordinator and worker agents.
type Router struct {
	pool    *WorkerPool
	mailbox *Mailbox // may be nil if no filesystem mailbox is configured
}

// NewRouter creates a router backed by the given worker pool.
// Pass a non-nil Mailbox to enable filesystem-based delivery as a fallback.
func NewRouter(pool *WorkerPool, mailbox *Mailbox) *Router {
	return &Router{pool: pool, mailbox: mailbox}
}

// RouteMessage delivers msg from one agent to a target.
//
// Target formats:
//   - agent ID   (e.g. "a-worker-abc123def4567890") → in-process channel delivery
//   - agent name (e.g. "researcher-1")              → resolved via pool name index
//   - "*"                                           → broadcast to all workers
//   - "uds:<path>"                                  → Unix domain socket (stub)
//   - "bridge:<id>"                                 → bridge connection (stub)
func (r *Router) RouteMessage(from types.AgentId, to string, msg types.Message) error {
	kind, target := classifyTarget(to)
	switch kind {
	case RouteInProcess:
		return r.inProcessRoute(target, msg)
	case RouteMailbox:
		return r.mailboxRoute(target, msg)
	case RouteUDS:
		return r.udsRoute(target, msg)
	case RouteBridge:
		return r.bridgeRoute(target, msg)
	default:
		return fmt.Errorf("unknown route kind for target %q", to)
	}
}

func (r *Router) inProcessRoute(target string, msg types.Message) error {
	if target == "*" {
		r.pool.BroadcastMessage(msg)
		return nil
	}

	// Try as agent ID first.
	if id, ok := types.ToAgentId(target); ok {
		w, found := r.pool.Get(id)
		if !found {
			return r.fallbackMailbox(id, msg)
		}
		w.Send(msg)
		return nil
	}

	// Try as name.
	w, found := r.pool.GetByName(target)
	if !found {
		return fmt.Errorf("agent %q not found in pool", target)
	}
	w.Send(msg)
	return nil
}

func (r *Router) mailboxRoute(target string, msg types.Message) error {
	if r.mailbox == nil {
		return fmt.Errorf("mailbox not configured")
	}
	return r.mailbox.Send(types.AgentId(target), msg)
}

func (r *Router) fallbackMailbox(id types.AgentId, msg types.Message) error {
	if r.mailbox == nil {
		return fmt.Errorf("agent %s not found and no mailbox configured", id)
	}
	return r.mailbox.Send(id, msg)
}

// udsRoute is a stub for Unix domain socket delivery.
func (r *Router) udsRoute(path string, _ types.Message) error {
	_ = path
	return fmt.Errorf("UDS route not yet implemented (path=%s)", path)
}

// bridgeRoute is a stub for bridge-based delivery.
func (r *Router) bridgeRoute(id string, _ types.Message) error {
	_ = id
	return fmt.Errorf("bridge route not yet implemented (id=%s)", id)
}

func classifyTarget(to string) (RouteKind, string) {
	if to == "*" {
		return RouteInProcess, "*"
	}
	if after, ok := strings.CutPrefix(to, "uds:"); ok {
		return RouteUDS, after
	}
	if after, ok := strings.CutPrefix(to, "bridge:"); ok {
		return RouteBridge, after
	}
	if _, ok := types.ToAgentId(to); ok {
		return RouteInProcess, to
	}
	// Names go through in-process first; mailbox is a fallback handled inside.
	return RouteInProcess, to
}
