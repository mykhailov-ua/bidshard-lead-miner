//go:build !unix

package tgweb

func withRegistrySharedLock(_ string, fn func() error) error {
	return fn()
}

func withRegistryExclusiveLock(_ string, fn func() error) error {
	return fn()
}
