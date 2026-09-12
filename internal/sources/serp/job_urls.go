package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/dorkdisable"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sources/jobboard"
)

const defaultJobboardRegistryPath = "data/runtime/discovered_job_urls.json"

// HarvestJobboardURLs runs DOU/Djinni SERP dorks and appends job-board URLs to the runtime registry.
func (c *Crawler) HarvestJobboardURLs(ctx context.Context, registryPath string) error {
	if registryPath == "" {
		registryPath = defaultJobboardRegistryPath
	}
	icpPath := discover.ResolveICPPath("")
	icp, err := discover.LoadICP(icpPath)
	if err != nil {
		slog.Warn("jobboard discover icp load failed", "path", icpPath, "error", err)
	}
	dorks := serpHarvestJobboardDorks(icp.SerpDorks)
	if len(dorks) == 0 {
		dorks = fallbackJobboardDorks()
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
			slog.Warn("jobboard discover serp failed", "dork", dork, "error", err)
			continue
		}
		items := ExtractJobboardDiscoveries(results)
		n, err := jobboard.AppendDiscoveries(registryPath, "serp", dork, items)
		if err != nil {
			slog.Warn("jobboard registry write failed", "error", err)
			continue
		}
		added += n
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("jobboard", added)
	}
	slog.Info("jobboard discover finished", "new_urls", added)
	return nil
}

func serpHarvestJobboardDorks(dorks []string) []string {
	all := serpHarvestDorksFromICP(dorks)
	out := make([]string, 0, len(all))
	for _, dork := range all {
		lower := strings.ToLower(dork)
		if strings.Contains(lower, "jobs.dou.ua") || strings.Contains(lower, "djinni.co") {
			out = append(out, dork)
		}
	}
	return out
}

func fallbackJobboardDorks() []string {
	return []string{
		`site:jobs.dou.ua "media buyer" keitaro`,
		`site:jobs.dou.ua "media buyer" gambling`,
		`site:djinni.co "media buyer" gambling`,
		`site:djinni.co "media buyer" keitaro`,
	}
}

// ExtractJobboardDiscoveries filters SERP hits to crawlable DOU/Djinni pages.
func ExtractJobboardDiscoveries(results []SERPResult) []jobboard.Discovery {
	seen := make(map[string]struct{})
	var out []jobboard.Discovery
	for _, res := range results {
		u := jobboard.NormalizeURL(res.URL)
		if u == "" || !jobboard.IsJobboardURL(u) {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, jobboard.Discovery{
			URL:     u,
			Title:   strings.TrimSpace(res.Title),
			Snippet: strings.TrimSpace(res.Snippet),
		})
	}
	return out
}
