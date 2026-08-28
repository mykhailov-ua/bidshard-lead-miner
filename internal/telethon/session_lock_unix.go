//go:build unix

package telethon

import (
	"os"
	"path/filepath"
	"syscall"
)

func withSessionExclusiveLock(sessionPath string, fn func() error) error {
	f, err := openSessionLock(sessionPath, syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer unlockSessionLock(f)
	return fn()
}

func openSessionLock(sessionPath string, how int) (*os.File, error) {
	lockPath := sessionLockPath(sessionPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func unlockSessionLock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}

func sessionLockPath(sessionPath string) string {
	return sessionPath + ".lock"
}
