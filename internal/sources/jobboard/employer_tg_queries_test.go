package jobboard

import (
	"path/filepath"
	"testing"
)

func TestAppendEmployerTGQueries(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "employer_tg_queries.json")
	added, err := AppendEmployerTGQueries(path, []string{"TapOK"}, "employer_reverse")
	if err != nil || added != 1 {
		t.Fatalf("added=%d err=%v", added, err)
	}
	added2, err := AppendEmployerTGQueries(path, []string{"TapOK", "TapOk"}, "employer_reverse")
	if err != nil || added2 != 0 {
		t.Fatalf("duplicate added=%d err=%v", added2, err)
	}
	f, err := LoadEmployerTGQueries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Queries) != 1 || f.Queries[0].Name != "TapOK" {
		t.Fatalf("queries=%v", f.Queries)
	}
}
