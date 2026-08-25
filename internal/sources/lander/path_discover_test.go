package lander

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

type stubPathFetcher map[string]stubResponse

type stubResponse struct {
	body   string
	status int
}

func (s stubPathFetcher) Get(_ context.Context, rawURL string) (string, int, error) {
	for key, resp := range s {
		if strings.Contains(rawURL, key) {
			return resp.body, resp.status, nil
		}
	}
	return "", 404, fmt.Errorf("not found: %s", rawURL)
}

func TestParseRobotsSitemaps(t *testing.T) {
	t.Parallel()
	body := "User-agent: *\nDisallow:\nSitemap: https://example.com/sitemap.xml\n"
	got := ParseRobotsSitemaps(body, "https://example.com/robots.txt")
	if len(got) != 1 || got[0] != "https://example.com/sitemap.xml" {
		t.Fatalf("got %v", got)
	}
}

func TestParseSitemapPageLocsDiscoversJoinPartners(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/join-partners</loc></url>
  <url><loc>https://example.com/blog/launch</loc></url>
</urlset>`
	paths := uniqueAffiliatePaths(func() []string {
		var out []string
		for _, loc := range ParseSitemapPageLocs(xml) {
			if p := URLToLandingPath(loc); p != "" {
				out = append(out, p)
			}
		}
		return out
	}())
	if len(paths) != 1 || paths[0] != "/join-partners" {
		t.Fatalf("paths=%v want /join-partners only", paths)
	}
}

func TestPathDiscovererFindsNonDefaultAffiliatePath(t *testing.T) {
	joinHTML := `<html><body><footer><p>Affiliate program and partner contact</p>` +
		`<a href="mailto:partners@affnet.com">partners@affnet.com</a></footer></body></html>`
	fetcher := stubPathFetcher{
		"/robots.txt": {body: "Sitemap: https://example.com/sitemap.xml\n", status: 200},
		"/sitemap.xml": {body: `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>https://example.com/join-partners</loc></url></urlset>`, status: 200},
		"/join-partners": {body: joinHTML, status: 200},
	}
	cachePath := t.TempDir() + "/lander_paths.json"
	d := NewPathDiscoverer(fetcher, PathDiscoverConfig{Enabled: true, CachePath: cachePath})
	paths := d.PathsForDomain(context.Background(), "example.com")

	foundJoin := false
	for _, p := range paths {
		if p == "/join-partners" {
			foundJoin = true
			break
		}
	}
	if !foundJoin {
		t.Fatalf("missing /join-partners in %v", paths)
	}
	for _, p := range NetworkLandingPaths() {
		if p == "/join-partners" {
			t.Fatal("default paths must not include /join-partners")
		}
	}

	// DoD: contact found on discovered path without /affiliate in seeds.
	pageURL := "https://example.com/join-partners"
	resp := fetcher["/join-partners"]
	text, _ := TextForContactExtract(resp.body)
	contacts := extract.Extract(text)
	if len(contacts.Contacts) == 0 {
		t.Fatalf("expected contact on %s text=%q", pageURL, text)
	}
}

func TestMergeLandingPathsDedupes(t *testing.T) {
	t.Parallel()
	got := MergeLandingPaths([]string{"/", "/about"}, []string{"/about", "/join-partners"})
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}
