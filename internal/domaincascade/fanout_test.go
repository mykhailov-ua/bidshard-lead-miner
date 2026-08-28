package domaincascade

import (
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/webpain"
)

func TestFanOutQueuesRegistryAndTgweb(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := Config{
		RegistryPath:        filepath.Join(dir, "source_registry.json"),
		TelegramDomainsPath: filepath.Join(dir, "tg_domains.json"),
	}

	regAdded, tgAdded, err := FanOut(cfg, "buylink.pro", "ads_txt", "supply")
	if err != nil {
		t.Fatal(err)
	}
	if !regAdded || !tgAdded {
		t.Fatalf("regAdded=%v tgAdded=%v", regAdded, tgAdded)
	}

	supply, err := sourceregistry.ListDomainsByType(cfg.RegistryPath, sourceregistry.TypeSupply)
	if err != nil {
		t.Fatal(err)
	}
	if len(supply) != 1 || supply[0] != "buylink.pro" {
		t.Fatalf("supply=%v", supply)
	}
	lander, err := sourceregistry.ListDomainsByType(cfg.RegistryPath, sourceregistry.TypeLander)
	if err != nil {
		t.Fatal(err)
	}
	if len(lander) != 1 {
		t.Fatalf("lander=%v", lander)
	}
}

func TestSyncDiscoveryRegistriesFromWebPain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	webPath := filepath.Join(dir, "web_pain.json")
	if _, err := webpain.AppendDiscoveries(webPath, "serp", "q", []webpain.Discovery{
		{URL: "https://tracker.example.com/blog/postback-fail"},
	}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RegistryPath:        filepath.Join(dir, "source_registry.json"),
		TelegramDomainsPath: filepath.Join(dir, "tg_domains.json"),
		WebPainRegistryPath: webPath,
	}
	regAdded, _, err := SyncDiscoveryRegistries(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if regAdded != 1 {
		t.Fatalf("regAdded=%d want 1", regAdded)
	}
}

func TestSyncDiscoveryRegistriesForumSnippet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	forumPath := filepath.Join(dir, "forum.json")
	if _, err := forum.AppendThreadDiscoveries(forumPath, "serp", "q", []forum.ThreadDiscovery{
		{
			URL:     "https://affiliatefix.com/threads/x",
			Snippet: "voluum alternative see https://network.io/partners",
		},
	}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RegistryPath:        filepath.Join(dir, "source_registry.json"),
		TelegramDomainsPath: filepath.Join(dir, "tg_domains.json"),
		ForumRegistryPath:   forumPath,
	}
	regAdded, _, err := SyncDiscoveryRegistries(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if regAdded != 1 {
		t.Fatalf("regAdded=%d want 1", regAdded)
	}
	domains, err := sourceregistry.ListDomainsByType(cfg.RegistryPath, sourceregistry.TypeSupply)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "network.io" {
		t.Fatalf("domains=%v", domains)
	}
}
