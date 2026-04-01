package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const staleLockThreshold = 30 * time.Second

// FileLock provides advisory file locking for config files.
type FileLock struct {
	path string
	f    *os.File
}

// Lock acquires an advisory lock by creating an exclusive lock file.
// Stale locks older than 30 seconds are automatically removed.
func Lock(path string) (*FileLock, error) {
	lockPath := path + ".lock"
	os.MkdirAll(filepath.Dir(lockPath), 0o755)

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		return &FileLock{path: lockPath, f: f}, nil
	}
	if !os.IsExist(err) {
		return nil, err
	}

	info, statErr := os.Stat(lockPath)
	if statErr != nil || time.Since(info.ModTime()) <= staleLockThreshold {
		return nil, fmt.Errorf("config file is locked: %s", lockPath)
	}

	os.Remove(lockPath)
	f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("config file is locked: %s", lockPath)
	}
	return &FileLock{path: lockPath, f: f}, nil
}

// Unlock releases the lock and removes the lock file.
func (l *FileLock) Unlock() error {
	if l.f != nil {
		l.f.Close()
	}
	return os.Remove(l.path)
}
