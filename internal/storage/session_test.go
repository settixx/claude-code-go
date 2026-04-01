package storage

import (
	"testing"
	"time"

	"github.com/settixx/claude-code-go/internal/types"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)

	sessionID := types.SessionId("test-session-001")
	messages := []types.Message{
		{
			Type:      types.MsgUser,
			UUID:      "uuid-1",
			Timestamp: time.Now().Truncate(time.Millisecond),
			Text:      "hello world",
		},
		{
			Type:      types.MsgAssistant,
			UUID:      "uuid-2",
			Timestamp: time.Now().Truncate(time.Millisecond),
			Text:      "hi there",
		},
	}

	if err := fs.Save(sessionID, messages); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := fs.Load(sessionID)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(loaded) != len(messages) {
		t.Fatalf("loaded %d messages, want %d", len(loaded), len(messages))
	}

	if loaded[0].UUID != "uuid-1" {
		t.Errorf("loaded[0].UUID = %q, want %q", loaded[0].UUID, "uuid-1")
	}
	if loaded[0].Text != "hello world" {
		t.Errorf("loaded[0].Text = %q, want %q", loaded[0].Text, "hello world")
	}
	if loaded[1].Type != types.MsgAssistant {
		t.Errorf("loaded[1].Type = %q, want %q", loaded[1].Type, types.MsgAssistant)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)

	for _, id := range []string{"session-a", "session-b"} {
		msgs := []types.Message{{
			Type: types.MsgUser,
			UUID: "u-" + id,
			Text: "msg for " + id,
		}}
		if err := fs.Save(types.SessionId(id), msgs); err != nil {
			t.Fatalf("Save(%s) error: %v", id, err)
		}
	}

	infos, err := fs.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}

	if len(infos) != 2 {
		t.Fatalf("List returned %d sessions, want 2", len(infos))
	}
}

func TestListEmptyDir(t *testing.T) {
	dir := t.TempDir() + "/nonexistent"
	fs := NewFileStorage(dir)

	infos, err := fs.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if infos != nil {
		t.Errorf("List on nonexistent dir should return nil, got %v", infos)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)

	sessionID := types.SessionId("to-delete")
	msgs := []types.Message{{Type: types.MsgUser, UUID: "u1", Text: "bye"}}
	if err := fs.Save(sessionID, msgs); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	if err := fs.Delete(sessionID); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	_, err := fs.Load(sessionID)
	if err == nil {
		t.Error("Load after Delete should return error")
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)

	_, err := fs.Load(types.SessionId("does-not-exist"))
	if err == nil {
		t.Error("Load of nonexistent session should return error")
	}
}
