//go:build unix

package telethon

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSessionExclusiveLockBlocksConcurrent(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "telethon.session")

	var wg sync.WaitGroup
	acquired := make(chan struct{}, 1)
	release := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := withSessionExclusiveLock(session, func() error {
			acquired <- struct{}{}
			<-release
			return nil
		})
		if err != nil {
			t.Errorf("holder: %v", err)
		}
	}()

	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("lock holder did not start")
	}

	done := make(chan error, 1)
	go func() {
		done <- withSessionExclusiveLock(session, func() error {
			return nil
		})
	}()

	select {
	case err := <-done:
		t.Fatalf("second lock acquired while holder active: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	wg.Wait()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second lock after release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second lock did not acquire after release")
	}

	lockPath := sessionLockPath(session)
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
}
