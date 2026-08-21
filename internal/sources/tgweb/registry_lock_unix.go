//go:build unix

package tgweb

import (
	"os"
	"path/filepath"
	"syscall"
)

func withRegistrySharedLock(path string, fn func() error) error {
	if path == "" {
		path = DefaultDomainsPath
	}
	f, err := openRegistryLock(path, syscall.LOCK_SH)
	if err != nil {
		return err
	}
	defer unlockRegistryLock(f)
	return fn()
}

func withRegistryExclusiveLock(path string, fn func() error) error {
	if path == "" {
		path = DefaultDomainsPath
	}
	f, err := openRegistryLock(path, syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer unlockRegistryLock(f)
	return fn()
}

func openRegistryLock(path string, how int) (*os.File, error) {
	// Sidecar lock file; shared for LoadDomains, exclusive for RMW writes (Go tgweb + Python Telethon).
	lockPath := registryLockPath(path)
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

func unlockRegistryLock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}

func registryLockPath(path string) string {
	return path + ".lock"
}
