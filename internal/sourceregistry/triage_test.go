package sourceregistry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldSkipCrawlBlocksPendingRegistryDomain(t *testing.T) {
	dir := t.TempDir()
	regPath := filepath.Join(dir, "registry.json")
	if _, err := Upsert(regPath, Entry{
		Domain:       "pending.example",
		Types:        []string{TypeTGWeb},
		DiscoveredBy: "telegram",
	}); err != nil {
		t.Fatal(err)
	}

	skip, reason := ShouldSkipCrawl(regPath, true, DomainMeta{Domain: "pending.example"})
	if !skip || reason != "triage:pending" {
		t.Fatalf("skip=%v reason=%q", skip, reason)
	}

	if err := SetTriageStatus(regPath, "pending.example", "keep"); err != nil {
		t.Fatal(err)
	}
	skip, _ = ShouldSkipCrawl(regPath, true, DomainMeta{Domain: "pending.example"})
	if skip {
		t.Fatal("expected crawl allowed after keep")
	}
}

func TestShouldSkipCrawlHeuristicDropsSocial(t *testing.T) {
	skip, reason := ShouldSkipCrawl("", false, DomainMeta{Domain: "t.me"})
	if !skip || !strings.HasPrefix(reason, "heuristic:") {
		t.Fatalf("skip=%v reason=%q", skip, reason)
	}
}
