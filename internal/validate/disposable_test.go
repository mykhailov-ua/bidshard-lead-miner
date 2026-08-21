package validate

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadDisposableDomains(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "disposable.txt")
	if err := os.WriteFile(path, []byte("custom-disposable.test\n# comment\n\nfoo.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadDisposableDomains(path); err != nil {
		t.Fatal(err)
	}
	if !IsDisposableDomain("custom-disposable.test") {
		t.Fatal("expected domain in disposable set")
	}
	if AcceptEmail("buyer@custom-disposable.test") {
		t.Fatal("expected loaded domain to be rejected")
	}
}

func TestDisposableDomainsConcurrentLoadAndRead(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "disposable.txt")
	if err := os.WriteFile(path, []byte("race-disposable.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = LoadDisposableDomains(path)
		}()
	}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = IsDisposableDomain("race-disposable.test")
			_ = AcceptEmail("ops@race-disposable.test")
			_ = DisposableDomainCount()
		}()
	}
	wg.Wait()
}
