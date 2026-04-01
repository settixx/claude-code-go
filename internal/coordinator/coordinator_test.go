package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// Worker lifecycle
// ---------------------------------------------------------------------------

func TestWorker_IdleToRunningToStopped(t *testing.T) {
	w := NewWorker("a-test-1234567890abcdef", "w1", "do stuff")
	if w.Status != WorkerIdle {
		t.Fatalf("initial status = %q, want idle", w.Status)
	}

	ctx := context.Background()
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if w.Status != WorkerRunning {
		t.Errorf("after start: status = %q, want running", w.Status)
	}

	w.Stop()
	<-w.Done()
	if w.Status != WorkerStopped {
		t.Errorf("after stop: status = %q, want stopped", w.Status)
	}
}

func TestWorker_DoubleStartFails(t *testing.T) {
	w := NewWorker("a-test-1234567890abcdef", "w1", "task")
	ctx := context.Background()
	_ = w.Start(ctx)
	defer w.Stop()

	err := w.Start(ctx)
	if err == nil {
		t.Error("expected error on double start")
	}
}

func TestWorker_SendAndReceiveMessages(t *testing.T) {
	w := NewWorker("a-test-1234567890abcdef", "w1", "task")
	ctx := context.Background()
	_ = w.Start(ctx)

	msg := types.Message{Type: types.MsgUser, Text: "hello"}
	w.Send(msg)

	time.Sleep(50 * time.Millisecond)

	msgs := w.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Text != "hello" {
		t.Errorf("text = %q", msgs[0].Text)
	}

	w.Stop()
}

func TestWorker_RunFuncOverridesLoop(t *testing.T) {
	w := NewWorker("a-test-1234567890abcdef", "w1", "task")
	w.RunFunc = func(_ context.Context, w *Worker) error {
		w.Result = "custom result"
		return nil
	}

	ctx := context.Background()
	_ = w.Start(ctx)
	<-w.Done()

	if w.Status != WorkerComplete {
		t.Errorf("status = %q, want complete", w.Status)
	}
	if w.Result != "custom result" {
		t.Errorf("result = %q", w.Result)
	}
}

func TestWorker_RunFuncError(t *testing.T) {
	w := NewWorker("a-test-1234567890abcdef", "w1", "task")
	w.RunFunc = func(_ context.Context, _ *Worker) error {
		return fmt.Errorf("boom")
	}

	ctx := context.Background()
	_ = w.Start(ctx)
	<-w.Done()

	if w.Status != WorkerFailed {
		t.Errorf("status = %q, want failed", w.Status)
	}
	if w.Err == nil || w.Err.Error() != "boom" {
		t.Errorf("err = %v", w.Err)
	}
}

func TestWorker_OnMessageHandler(t *testing.T) {
	var received []string
	w := NewWorker("a-test-1234567890abcdef", "w1", "task")
	w.OnMessage = func(_ context.Context, _ *Worker, msg types.Message) error {
		received = append(received, msg.Text)
		return nil
	}

	ctx := context.Background()
	_ = w.Start(ctx)

	w.Send(types.Message{Type: types.MsgUser, Text: "msg1"})
	w.Send(types.Message{Type: types.MsgUser, Text: "msg2"})
	time.Sleep(50 * time.Millisecond)

	w.Stop()

	if len(received) != 2 {
		t.Errorf("handler received %d messages, want 2", len(received))
	}
}

// ---------------------------------------------------------------------------
// WorkerPool
// ---------------------------------------------------------------------------

func TestWorkerPool_SpawnAndGet(t *testing.T) {
	pool := NewWorkerPool()
	ctx := context.Background()

	w, err := pool.SpawnWorker(ctx, "alpha", "do alpha", nil)
	if err != nil {
		t.Fatalf("SpawnWorker: %v", err)
	}

	got, ok := pool.Get(w.ID)
	if !ok {
		t.Fatal("worker not found by ID")
	}
	if got.Name != "alpha" {
		t.Errorf("name = %q", got.Name)
	}

	byName, ok := pool.GetByName("alpha")
	if !ok {
		t.Fatal("worker not found by name")
	}
	if byName.ID != w.ID {
		t.Error("ID mismatch")
	}

	pool.StopAll()
}

func TestWorkerPool_StopWorker(t *testing.T) {
	pool := NewWorkerPool()
	ctx := context.Background()

	w, _ := pool.SpawnWorker(ctx, "temp", "task", nil)
	id := w.ID

	if err := pool.StopWorker(id); err != nil {
		t.Fatalf("StopWorker: %v", err)
	}

	_, ok := pool.Get(id)
	if ok {
		t.Error("worker should be removed after stop")
	}
}

func TestWorkerPool_StopWorker_NotFound(t *testing.T) {
	pool := NewWorkerPool()
	err := pool.StopWorker("a-nonexistent-1234567890abcdef")
	if err == nil {
		t.Error("expected error for nonexistent worker")
	}
}

func TestWorkerPool_BroadcastMessage(t *testing.T) {
	pool := NewWorkerPool()
	ctx := context.Background()

	w1, _ := pool.SpawnWorker(ctx, "w1", "t1", nil)
	w2, _ := pool.SpawnWorker(ctx, "w2", "t2", nil)

	pool.BroadcastMessage(types.Message{Type: types.MsgUser, Text: "broadcast"})
	time.Sleep(50 * time.Millisecond)

	for _, w := range []*Worker{w1, w2} {
		msgs := w.Messages()
		if len(msgs) == 0 {
			t.Errorf("worker %q got no broadcast", w.Name)
		}
	}

	pool.StopAll()
}

func TestWorkerPool_All(t *testing.T) {
	pool := NewWorkerPool()
	ctx := context.Background()

	pool.SpawnWorker(ctx, "a", "t", nil)
	pool.SpawnWorker(ctx, "b", "t", nil)

	all := pool.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d, want 2", len(all))
	}

	pool.StopAll()
}

// ---------------------------------------------------------------------------
// TeamManager
// ---------------------------------------------------------------------------

func TestTeamManager_CreateAndGet(t *testing.T) {
	pool := NewWorkerPool()
	tm := NewTeamManager(pool)

	members := []types.AgentId{"a-m1-1234567890abcdef", "a-m2-1234567890abcdef"}
	team, err := tm.Create("my-team", members)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if team.Name != "my-team" {
		t.Errorf("name = %q", team.Name)
	}
	if team.Leader != members[0] {
		t.Errorf("leader = %q, want %q", team.Leader, members[0])
	}

	got, ok := tm.Get("my-team")
	if !ok {
		t.Fatal("team not found")
	}
	if len(got.Members) != 2 {
		t.Errorf("members = %d", len(got.Members))
	}
}

func TestTeamManager_CreateDuplicate(t *testing.T) {
	pool := NewWorkerPool()
	tm := NewTeamManager(pool)

	members := []types.AgentId{"a-m1-1234567890abcdef"}
	tm.Create("dup", members)
	_, err := tm.Create("dup", members)
	if err == nil {
		t.Error("expected error for duplicate team name")
	}
}

func TestTeamManager_CreateEmptyMembers(t *testing.T) {
	pool := NewWorkerPool()
	tm := NewTeamManager(pool)

	_, err := tm.Create("empty", nil)
	if err == nil {
		t.Error("expected error for empty members")
	}
}

func TestTeamManager_Delete(t *testing.T) {
	pool := NewWorkerPool()
	tm := NewTeamManager(pool)

	tm.Create("del-me", []types.AgentId{"a-m1-1234567890abcdef"})
	if err := tm.Delete("del-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok := tm.Get("del-me")
	if ok {
		t.Error("team should be deleted")
	}
}

func TestTeamManager_DeleteNotFound(t *testing.T) {
	pool := NewWorkerPool()
	tm := NewTeamManager(pool)

	err := tm.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
}

func TestTeamManager_List(t *testing.T) {
	pool := NewWorkerPool()
	tm := NewTeamManager(pool)

	tm.Create("a", []types.AgentId{"a-m1-1234567890abcdef"})
	tm.Create("b", []types.AgentId{"a-m2-1234567890abcdef"})

	names := tm.List()
	if len(names) != 2 {
		t.Errorf("List() = %d, want 2", len(names))
	}
}

// ---------------------------------------------------------------------------
// AgentTask state machine
// ---------------------------------------------------------------------------

func TestAgentTask_Lifecycle(t *testing.T) {
	w := NewWorker("a-task-1234567890abcdef", "w", "p")
	task := NewAgentTask("task-1", types.TaskLocalAgent, "my task", w)

	state := task.State()
	if state.Status != types.TaskPending {
		t.Fatalf("initial status = %q", state.Status)
	}

	if err := task.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if task.State().Status != types.TaskRunning {
		t.Error("should be running")
	}

	if err := task.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if task.State().Status != types.TaskComplete {
		t.Error("should be complete")
	}
}

func TestAgentTask_CannotStartTwice(t *testing.T) {
	w := NewWorker("a-task-1234567890abcdef", "w", "p")
	task := NewAgentTask("task-1", types.TaskLocalAgent, "t", w)
	task.Start()

	err := task.Start()
	if err == nil {
		t.Error("expected error on double start")
	}
}

func TestAgentTask_FailPath(t *testing.T) {
	w := NewWorker("a-task-1234567890abcdef", "w", "p")
	task := NewAgentTask("task-1", types.TaskLocalAgent, "t", w)
	task.Start()

	if err := task.Fail(); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if task.State().Status != types.TaskFailed {
		t.Error("should be failed")
	}
}

func TestAgentTask_StopPath(t *testing.T) {
	w := NewWorker("a-task-1234567890abcdef", "w", "p")
	task := NewAgentTask("task-1", types.TaskLocalAgent, "t", w)
	task.Start()

	if err := task.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if task.State().Status != types.TaskStopped {
		t.Error("should be stopped")
	}
}

func TestAgentTask_Progress(t *testing.T) {
	w := NewWorker("a-task-1234567890abcdef", "w", "p")
	task := NewAgentTask("task-1", types.TaskLocalAgent, "t", w)

	task.NotifyProgress("step 1")
	task.NotifyProgress("step 2")

	progress := task.Progress()
	if len(progress) != 2 {
		t.Fatalf("got %d entries, want 2", len(progress))
	}
	if progress[0].Message != "step 1" {
		t.Errorf("first = %q", progress[0].Message)
	}
}

// ---------------------------------------------------------------------------
// Mailbox
// ---------------------------------------------------------------------------

func TestMailbox_SendAndReceive(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)
	agentID := types.AgentId("a-test-1234567890abcdef")

	msg := types.Message{Type: types.MsgUser, Text: "hello"}
	if err := mb.Send(agentID, msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	msgs, err := mb.Receive(agentID)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages", len(msgs))
	}
	if msgs[0].Text != "hello" {
		t.Errorf("text = %q", msgs[0].Text)
	}

	// Second receive should be empty (messages consumed)
	msgs2, _ := mb.Receive(agentID)
	if len(msgs2) != 0 {
		t.Errorf("expected empty after consume, got %d", len(msgs2))
	}
}

func TestMailbox_ReceiveEmpty(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)

	msgs, err := mb.Receive("a-noone-1234567890abcdef")
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty, got %d", len(msgs))
	}
}

func TestMailbox_MessageOrdering(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)
	agentID := types.AgentId("a-order-1234567890abcdef")

	for i := 0; i < 5; i++ {
		mb.Send(agentID, types.Message{Text: fmt.Sprintf("msg-%d", i)})
		time.Sleep(time.Millisecond)
	}

	msgs, _ := mb.Receive(agentID)
	if len(msgs) != 5 {
		t.Fatalf("got %d", len(msgs))
	}
	for i, m := range msgs {
		want := fmt.Sprintf("msg-%d", i)
		if m.Text != want {
			t.Errorf("[%d] = %q, want %q", i, m.Text, want)
		}
	}
}

func TestMailbox_MessageIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)
	agentID := types.AgentId("a-json-1234567890abcdef")

	mb.Send(agentID, types.Message{Type: types.MsgUser, Text: "test"})

	agentDir := filepath.Join(dir, string(agentID))
	entries, _ := os.ReadDir(agentDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	data, _ := os.ReadFile(filepath.Join(agentDir, entries[0].Name()))
	var msg types.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Coordinator mode detection
// ---------------------------------------------------------------------------

func TestIsCoordinatorMode(t *testing.T) {
	if IsCoordinatorMode(types.AppState{Agent: "coordinator"}) != true {
		t.Error("should be coordinator mode")
	}
	if IsCoordinatorMode(types.AppState{Agent: ""}) != false {
		t.Error("should not be coordinator mode")
	}
}

func TestGetCoordinatorSystemPrompt(t *testing.T) {
	prompt := GetCoordinatorSystemPrompt()
	sections := []string{
		"Multi-Agent Coordinator",
		"Execution Phases",
		"Available Tools",
		"Constraints",
	}
	for _, s := range sections {
		if !contains(prompt, s) {
			t.Errorf("prompt missing section %q", s)
		}
	}
}

func TestParseWorktreeList(t *testing.T) {
	raw := `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/wt-feature
HEAD def456
branch refs/heads/feature
`
	result := parseWorktreeList(raw)
	if len(result) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(result))
	}
	if result[0].Path != "/home/user/project" {
		t.Errorf("path[0] = %q", result[0].Path)
	}
	if result[1].Branch != "refs/heads/feature" {
		t.Errorf("branch[1] = %q", result[1].Branch)
	}
}

// ---------------------------------------------------------------------------
// Router target classification
// ---------------------------------------------------------------------------

func TestClassifyTarget(t *testing.T) {
	tests := []struct {
		to       string
		wantKind RouteKind
	}{
		{"*", RouteInProcess},
		{"uds:/tmp/sock", RouteUDS},
		{"bridge:abc", RouteBridge},
		{"a-name-1234567890abcdef", RouteInProcess},
		{"some-name", RouteInProcess},
	}
	for _, tt := range tests {
		kind, _ := classifyTarget(tt.to)
		if kind != tt.wantKind {
			t.Errorf("classifyTarget(%q) = %q, want %q", tt.to, kind, tt.wantKind)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
