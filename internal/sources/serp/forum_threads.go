package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sources/forum"
)

const defaultForumThreadsPath = "data/runtime/discovered_forum_threads.json"

// HarvestForumThreads runs forum-targeted SERP dorks and appends thread URLs to the runtime registry.
func (c *Crawler) HarvestForumThreads(ctx context.Context, registryPath string) error {
	if registryPath == "" {
		registryPath = defaultForumThreadsPath
	}
	icpPath := discover.ResolveICPPath("")
	icp, err := discover.LoadICP(icpPath)
	if err != nil {
		slog.Warn("forum discover icp load failed, using fallback dorks", "path", icpPath, "error", err)
	}
	dorks := forumDorksFromICP(icp.SerpDorks)
	if len(dorks) == 0 {
		dorks = fallbackForumDorks()
	}

	var added int
	for _, dork := range dorks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		results, err := c.searchDork(ctx, dork)
		if err != nil {
			slog.Warn("forum discover serp failed", "dork", dork, "error", err)
			continue
		}
		items := ExtractForumThreadDiscoveries(results)
		n, err := forum.AppendThreadDiscoveries(registryPath, "serp", dork, items)
		if err != nil {
			slog.Warn("forum thread registry write failed", "error", err)
			continue
		}
		added += n
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("forum", added)
	}
	slog.Info("forum thread discover finished", "new_threads", added)
	return nil
}

func forumDorksFromICP(dorks []string) []string {
	var out []string
	for _, dork := range dorks {
		lower := strings.ToLower(dork)
		if strings.Contains(lower, "site:affiliatefix.com") ||
			strings.Contains(lower, "site:stmforum.com") ||
			strings.Contains(lower, "site:blackhatworld.com") ||
			strings.Contains(lower, "site:warriorforum.com") ||
			strings.Contains(lower, "site:afflift.com") {
			out = append(out, dork)
		}
	}
	return out
}

func fallbackForumDorks() []string {
	return []string{
		`site:affiliatefix.com "voluum alternative"`,
		`site:affiliatefix.com "postback failing"`,
		`site:stmforum.com tracker`,
		`site:blackhatworld.com "voluum alternative"`,
		`site:warriorforum.com "self-hosted tracker"`,
	}
}

// ExtractForumThreadURLs filters SERP hits down to crawlable forum thread URLs.
func ExtractForumThreadURLs(results []SERPResult) []string {
	items := ExtractForumThreadDiscoveries(results)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.URL)
	}
	return out
}

// ExtractForumThreadDiscoveries filters SERP hits and keeps title/snippet for triage.
func ExtractForumThreadDiscoveries(results []SERPResult) []forum.ThreadDiscovery {
	seen := make(map[string]struct{})
	var out []forum.ThreadDiscovery
	for _, res := range results {
		u := strings.TrimSpace(res.URL)
		if u == "" || !forum.IsForumThreadURL(u) {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, forum.ThreadDiscovery{
			URL:     u,
			Title:   strings.TrimSpace(res.Title),
			Snippet: strings.TrimSpace(res.Snippet),
		})
	}
	return out
}
