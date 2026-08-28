package dorkdisable

import (
	"path/filepath"
	"testing"
)

func TestFilterActiveDorks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "disabled.json")
	if err := Save(path, []string{`site:t.me junk`}, []string{"test"}); err != nil {
		t.Fatal(err)
	}
	in := []string{`site:t.me good`, `site:t.me junk`, `site:t.me other`}
	got := FilterActiveDorks(path, in)
	if len(got) != 2 {
		t.Fatalf("got=%v", got)
	}
	if IsDisabled(path, `site:t.me junk`) != true {
		t.Fatal("expected disabled")
	}
}
