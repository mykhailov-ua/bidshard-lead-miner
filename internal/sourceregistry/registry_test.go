package sourceregistry

import (
	"path/filepath"
	"testing"
)

func TestUpsertMergesTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source_registry.json")

	if _, err := Upsert(path, Entry{Domain: "buylink.pro", Types: []string{TypeTGWeb}, DiscoveredBy: "telegram"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(path, Entry{Domain: "buylink.pro", Types: []string{TypeSupply}, DiscoveredBy: "telegram"}); err != nil {
		t.Fatal(err)
	}

	got, err := ListDomainsByType(path, TypeSupply)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "buylink.pro" {
		t.Fatalf("supply domains=%v", got)
	}
}

func TestListDomainsByTypeSkipsDrop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	if _, err := Upsert(path, Entry{
		Domain:       "junk.example",
		Types:        []string{TypeSupply},
		DiscoveredBy: "serp",
		TriageStatus: "drop",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := ListDomainsByType(path, TypeSupply)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got=%v want empty", got)
	}
}
