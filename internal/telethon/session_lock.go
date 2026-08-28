package telethon

func runWithSessionLock(configPath string, fn func() error) error {
	return withSessionExclusiveLock(SessionPath(configPath), fn)
}
