package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotateQueriesFromICPCycles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	icpPath := filepath.Join(dir, "discover.icp.json")
	statePath := filepath.Join(dir, "rotate.json")
	icp := `{"telegram_search":["voluum alternative","binom migration","keitaro api"],"serp_dorks":[]}`
	if err := os.WriteFile(icpPath, []byte(icp), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := RotateQueriesFromICP(icpPath, statePath, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 {
		t.Fatalf("first=%v", first)
	}
	second, err := RotateQueriesFromICP(icpPath, statePath, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 2 {
		t.Fatalf("second=%v", second)
	}
	if first[0] == second[0] && first[1] == second[1] {
		t.Fatalf("expected rotation first=%v second=%v", first, second)
	}
}
