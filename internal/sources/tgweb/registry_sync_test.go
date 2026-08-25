package tgweb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/sourceregistry"
)

func TestSyncToSourceRegistryFansOutSupply(t *testing.T) {
	dir := t.TempDir()
	tgPath := filepath.Join(dir, "tg.json")
	regPath := filepath.Join(dir, "source_registry.json")

	raw, err := json.Marshal(DomainFile{
		Domains: []DomainEntry{
			{Domain: "buylink.pro", Channel: "aff_net", Source: "cross_mention", At: "2026-01-01T00:00:00Z"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tgPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	added, err := SyncToSourceRegistry(tgPath, regPath)
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}

	supply, err := sourceregistry.ListDomainsByType(regPath, sourceregistry.TypeSupply)
	if err != nil {
		t.Fatal(err)
	}
	if len(supply) != 1 || supply[0] != "buylink.pro" {
		t.Fatalf("supply=%v", supply)
	}
}
