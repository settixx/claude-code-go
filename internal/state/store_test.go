package state

import (
	"sync"
	"testing"

	"github.com/settixx/claude-code-go/internal/types"
)

func TestStoreGetReturnsInitialState(t *testing.T) {
	initial := types.AppState{
		MainLoopModel: "test-model",
		Verbose:       true,
	}
	s := NewStore(initial)

	got := s.Get()
	if got.MainLoopModel != "test-model" {
		t.Errorf("MainLoopModel = %q, want %q", got.MainLoopModel, "test-model")
	}
	if !got.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestStoreUpdateModifiesState(t *testing.T) {
	s := NewStore(types.AppState{MainLoopModel: "old"})

	s.Update(func(st *types.AppState) {
		st.MainLoopModel = "new"
		st.Verbose = true
	})

	got := s.Get()
	if got.MainLoopModel != "new" {
		t.Errorf("MainLoopModel = %q, want %q", got.MainLoopModel, "new")
	}
	if !got.Verbose {
		t.Error("Verbose should be true after update")
	}
}

func TestStoreSubscribeFiresOnUpdate(t *testing.T) {
	s := NewStore(types.AppState{})

	var received types.AppState
	called := false
	unsub := s.Subscribe(func(st types.AppState) {
		called = true
		received = st
	})
	defer unsub()

	s.Update(func(st *types.AppState) {
		st.StatusLineText = "working..."
	})

	if !called {
		t.Fatal("subscriber was not called")
	}
	if received.StatusLineText != "working..." {
		t.Errorf("StatusLineText = %q, want %q", received.StatusLineText, "working...")
	}
}

func TestStoreUnsubscribeStopsCallbacks(t *testing.T) {
	s := NewStore(types.AppState{})

	callCount := 0
	unsub := s.Subscribe(func(_ types.AppState) {
		callCount++
	})

	s.Update(func(st *types.AppState) { st.Verbose = true })
	unsub()
	s.Update(func(st *types.AppState) { st.Verbose = false })

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should stop after unsub)", callCount)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := NewStore(types.AppState{})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Update(func(st *types.AppState) {
				st.Verbose = !st.Verbose
			})
			_ = s.Get()
		}()
	}
	wg.Wait()
}
