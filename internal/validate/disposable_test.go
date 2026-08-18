package validate

import (
	"os"
	"path/filepath"
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
