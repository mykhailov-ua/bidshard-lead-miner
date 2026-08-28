package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/dorkdisable"
	"github.com/bidshard/parser/internal/domaincascade"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/webpain"
)

const defaultWebPainPath = "data/runtime/discovered_web_pain.json"

// HarvestWebPainCatalog runs open-web SERP dorks (no site: operator) and appends page URLs.
func (c *Crawler) HarvestWebPainCatalog(ctx context.Context, registryPath string, cascade domaincascade.Config) error {
	if registryPath == "" {
		registryPath = defaultWebPainPath
	}
	icpPath := discover.ResolveICPPath("")
	icp, err := discover.LoadICP(icpPath)
	if err != nil {
		slog.Warn("web pain icp load failed", "path", icpPath, "error", err)
	}
	dorks := serpHarvestOpenWebDorks(icp.SerpDorks)
	if len(dorks) == 0 {
		dorks = fallbackOpenWebDorks()
	}
	dorks = dorkdisable.FilterActiveDorks(c.disabledDorksPath, dorks)

	var added int
	for _, dork := range dorks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		results, err := c.searchDork(ctx, dork)
		if err != nil {
			slog.Warn("web pain serp failed", "dork", dork, "error", err)
			continue
		}
		if cascade.RegistryPath != "" {
			var serpDomains []string
			for _, res := range results {
				d := strings.TrimSpace(res.Domain)
				if d == "" || isSkippedWebPainHost(d) {
					continue
				}
				serpDomains = append(serpDomains, d)
			}
			if len(serpDomains) > 0 {
				_, _, cascadeErr := domaincascade.FanOutMany(cascade, serpDomains, "serp", "webpain")
				if cascadeErr != nil {
					slog.Debug("web pain cascade failed", "error", cascadeErr)
				}
			}
		}
		items := ExtractWebPainDiscoveries(results)
		n, err := webpain.AppendDiscoveries(registryPath, "serp", dork, items)
		if err != nil {
			slog.Warn("web pain registry write failed", "error", err)
			continue
		}
		added += n
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("webpain", added)
	}
	slog.Info("web pain discover finished", "new_urls", added)
	return nil
}

// serpHarvestOpenWebDorks returns ICP dorks without a site: operator (open web search).
func serpHarvestOpenWebDorks(dorks []string) []string {
	all := serpHarvestDorksFromICP(dorks)
	out := make([]string, 0, len(all))
	for _, dork := range all {
		if strings.Contains(strings.ToLower(dork), "site:") {
			continue
		}
		out = append(out, dork)
	}
	return out
}

func fallbackOpenWebDorks() []string {
	return []string{
		`"postback failing" tracker`,
		`"voluum alternative" affiliate`,
		`"keitaro alternative" tracker`,
	}
}

// ExtractWebPainDiscoveries keeps SERP hits that are not forum thread URLs on allowlisted hosts.
func ExtractWebPainDiscoveries(results []SERPResult) []webpain.Discovery {
	seen := make(map[string]struct{})
	var out []webpain.Discovery
	for _, res := range results {
		u := strings.TrimSpace(res.URL)
		if u == "" {
			continue
		}
		domain := extractDomain(u)
		if domain == "" || isBlockedSERPDomain(domain) {
			continue
		}
		if forum.IsForumThreadURL(u) {
			continue
		}
		if isSkippedWebPainHost(domain) {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, webpain.Discovery{
			URL:     u,
			Title:   strings.TrimSpace(res.Title),
			Snippet: strings.TrimSpace(res.Snippet),
		})
	}
	return out
}

func isSkippedWebPainHost(domain string) bool {
	switch domain {
	case "reddit.com", "www.reddit.com", "old.reddit.com",
		"t.me", "telegram.me", "twitter.com", "x.com",
		"facebook.com", "www.facebook.com", "instagram.com", "www.instagram.com",
		"youtube.com", "www.youtube.com", "linkedin.com", "www.linkedin.com":
		return true
	default:
		return false
	}
}
