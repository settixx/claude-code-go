package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLock_BasicLockUnlock(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")

	lock, err := Lock(target)
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}

	lockFile := target + ".lock"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("lock file should exist")
	}

	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Error("lock file should be removed after unlock")
	}
}

func TestLock_DoubleLockFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")

	lock1, err := Lock(target)
	if err != nil {
		t.Fatalf("first Lock: %v", err)
	}
	defer lock1.Unlock()

	_, err = Lock(target)
	if err == nil {
		t.Error("second Lock should fail while first is held")
	}
}

func TestLock_StaleLockRemoved(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")

	lockFile := target + ".lock"
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("create stale lock: %v", err)
	}
	f.Close()

	// Set mod time to well beyond the stale threshold
	staleTime := time.Now().Add(-2 * staleLockThreshold)
	os.Chtimes(lockFile, staleTime, staleTime)

	lock, err := Lock(target)
	if err != nil {
		t.Fatalf("Lock with stale lock should succeed: %v", err)
	}
	lock.Unlock()
}

func TestLock_FreshLockNotRemoved(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")

	lockFile := target + ".lock"
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("create fresh lock: %v", err)
	}
	f.Close()

	// Mod time is now (fresh), so Lock should fail
	_, err = Lock(target)
	if err == nil {
		t.Error("Lock with fresh existing lock should fail")
	}

	// Clean up
	os.Remove(lockFile)
}

func TestLock_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "test.json")

	lock, err := Lock(deep)
	if err != nil {
		t.Fatalf("Lock with deep path: %v", err)
	}
	lock.Unlock()
}

// ---------------------------------------------------------------------------
// Lock — unlock is idempotent for lock file removal
// ---------------------------------------------------------------------------

func TestLock_UnlockRemovesFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data.json")

	lock, err := Lock(target)
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}

	lockPath := target + ".lock"
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("lock file should exist before unlock")
	}

	if err := lock.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should not exist after unlock")
	}
}

// ---------------------------------------------------------------------------
// Lock — re-lock after unlock
// ---------------------------------------------------------------------------

func TestLock_RelockAfterUnlock(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")

	lock1, err := Lock(target)
	if err != nil {
		t.Fatalf("first Lock: %v", err)
	}
	lock1.Unlock()

	lock2, err := Lock(target)
	if err != nil {
		t.Fatalf("second Lock after unlock should succeed: %v", err)
	}
	lock2.Unlock()
}

// ---------------------------------------------------------------------------
// Lock — stale threshold boundary
// ---------------------------------------------------------------------------

func TestLock_StaleBoundary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")
	lockFile := target + ".lock"

	f, _ := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	f.Close()

	justUnderStale := time.Now().Add(-staleLockThreshold + 5*time.Second)
	os.Chtimes(lockFile, justUnderStale, justUnderStale)

	_, err := Lock(target)
	if err == nil {
		t.Error("Lock should fail when lock is just under the stale threshold")
	}

	os.Remove(lockFile)
}
