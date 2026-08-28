//go:build !unix

package telethon

func withSessionExclusiveLock(_ string, fn func() error) error {
	return fn()
}
