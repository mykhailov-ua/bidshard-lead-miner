package telethon

import "sync"

var sessionProcMu sync.Mutex

func runWithSessionLock(configPath string, fn func() error) error {
	sessionProcMu.Lock()
	defer sessionProcMu.Unlock()
	return withSessionExclusiveLock(SessionPath(configPath), fn)
}
