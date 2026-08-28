package lander

import (
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/sourceregistry"
)

func TestLoadURLsCombinedUsesRegistry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	regPath := filepath.Join(dir, "source_registry.json")
	if _, err := sourceregistry.UpsertCascade(regPath, "example.com", "test", "supply", ""); err != nil {
		t.Fatal(err)
	}

	urls, err := LoadURLsCombined("", regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 {
		t.Fatalf("urls=%v", urls)
	}
	if urls[0] != "https://example.com/affiliate" {
		t.Fatalf("url=%q", urls[0])
	}
}

func TestPrimarySeedURLPrefersAffiliatePath(t *testing.T) {
	t.Parallel()

	got := PrimarySeedURL("acme.com")
	if got != "https://acme.com/affiliate" {
		t.Fatalf("got %q", got)
	}
}
