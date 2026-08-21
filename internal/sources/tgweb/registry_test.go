package tgweb

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPendingDomains(t *testing.T) {
	t.Parallel()
	f := DomainFile{
		Domains: []DomainEntry{
			{Domain: "fresh.com"},
			{Domain: "recent.com", CrawledAt: time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339)},
			{Domain: "13-02-2023-1.jpg"},
		},
	}
	got := PendingDomains(f, 10, 30)
	if len(got) != 1 || got[0].Domain != "fresh.com" {
		t.Fatalf("got %v", got)
	}
}

func TestPendingDomainsPrioritizesChannelAndRecent(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	f := DomainFile{
		Domains: []DomainEntry{
			{Domain: "old-nolink.com", At: now.AddDate(0, 0, -20).Format(time.RFC3339)},
			{Domain: "fresh-channel.com", Channel: "aff_ops", At: now.AddDate(0, 0, -1).Format(time.RFC3339)},
			{Domain: "fresh-nolink.com", At: now.AddDate(0, 0, -1).Format(time.RFC3339)},
		},
	}
	got := PendingDomains(f, 2, 30)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got[0].Domain != "fresh-channel.com" {
		t.Fatalf("first=%s want fresh-channel.com (channel+recent)", got[0].Domain)
	}
	if got[1].Domain != "fresh-nolink.com" {
		t.Fatalf("second=%s want fresh-nolink.com", got[1].Domain)
	}
}

func TestPruneInvalidDomains(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "domains.json")
	raw := `{
  "domains": [
    {"domain": "bojoko.com", "channel": "wooden_blog", "source": "discover"},
    {"domain": "13-02-2023-1.jpg", "channel": "x", "source": "discover"},
    {"domain": "google.co.uk", "channel": "x", "source": "discover"},
    {"domain": "durov.gram", "channel": "x", "source": "discover"}
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	kept, removed, err := PruneInvalidDomains(path)
	if err != nil {
		t.Fatal(err)
	}
	if kept != 1 || removed != 3 {
		t.Fatalf("kept=%d removed=%d", kept, removed)
	}

	f, err := LoadDomains(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Domains) != 1 || f.Domains[0].Domain != "bojoko.com" {
		t.Fatalf("domains=%v", f.Domains)
	}
}

func TestClearCrawledAt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "domains.json")
	raw := `{
  "domains": [
    {"domain": "blask.com", "crawled_at": "2026-08-19T21:59:18Z"},
    {"domain": "vietnam.vn", "crawled_at": "2026-08-19T21:59:28Z"},
    {"domain": "bojoko.com"}
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	updated, err := ClearCrawledAt(path, []string{"blask.com", "vietnam.vn"})
	if err != nil {
		t.Fatal(err)
	}
	if updated != 2 {
		t.Fatalf("updated=%d", updated)
	}

	f, err := LoadDomains(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Domains {
		if d.Domain == "blask.com" || d.Domain == "vietnam.vn" {
			if d.CrawledAt != "" {
				t.Fatalf("%s still crawled: %s", d.Domain, d.CrawledAt)
			}
		}
	}

	got := SelectDomains(f, 10, 30, []string{"blask.com", "vietnam.vn"})
	if len(got) != 2 {
		t.Fatalf("select only got %v", got)
	}
}

func TestPendingDomainsPrioritizesAboutMentionKind(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	at := now.AddDate(0, 0, -1).Format(time.RFC3339)
	f := DomainFile{
		Domains: []DomainEntry{
			{Domain: "msg-only.com", Channel: "aff_ops", At: at, Kind: "mentioned_in_message"},
			{Domain: "about.com", Channel: "aff_ops", At: at, Kind: "mentioned_in_about"},
		},
	}
	got := PendingDomains(f, 2, 30)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got[0].Domain != "about.com" {
		t.Fatalf("first=%s want about.com (mentioned_in_about)", got[0].Domain)
	}
}

func TestLoadDomainsNormalizesLegacyForwardedFrom(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "domains.json")
	raw := `{
  "domains": [
    {"domain": "affnet.com", "forwarded_from": "seed_chat", "kind": "mentioned_in_about"}
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := LoadDomains(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Domains) != 1 {
		t.Fatalf("domains=%v", f.Domains)
	}
	if f.Domains[0].DiscoveredVia != "seed_chat" {
		t.Fatalf("discovered_via=%q", f.Domains[0].DiscoveredVia)
	}
	if f.Domains[0].ForwardedFrom != "" {
		t.Fatalf("legacy forwarded_from should be cleared, got %q", f.Domains[0].ForwardedFrom)
	}
}

func TestSiteDomainFromSource(t *testing.T) {
	t.Parallel()

	if got := SiteDomainFromSource("tgweb:@wooden_blog:bojoko.com"); got != "bojoko.com" {
		t.Fatalf("got %q", got)
	}
	if got := SiteDomainFromSource("tgweb:bojoko.com"); got != "bojoko.com" {
		t.Fatalf("got %q", got)
	}
	if got := SiteDomainFromSource("stub:test"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestMarkCrawledConcurrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "domains.json")
	raw := `{
  "domains": [
    {"domain": "one.com", "source": "discover"},
    {"domain": "two.com", "source": "discover"}
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, domain := range []string{"one.com", "two.com", "one.com", "two.com"} {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			if err := MarkCrawled(path, d); err != nil {
				t.Errorf("MarkCrawled(%s): %v", d, err)
			}
		}(domain)
	}
	wg.Wait()

	f, err := LoadDomains(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"one.com", "two.com"} {
		found := false
		for _, e := range f.Domains {
			if e.Domain == want {
				found = true
				if e.CrawledAt == "" {
					t.Fatalf("%s missing crawled_at", want)
				}
			}
		}
		if !found {
			t.Fatalf("domain %s missing from registry", want)
		}
	}
}
