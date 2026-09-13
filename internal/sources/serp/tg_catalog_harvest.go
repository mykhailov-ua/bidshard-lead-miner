package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/dorkdisable"
	"github.com/bidshard/parser/internal/extract"
)

func fallbackTGCatalogMetaDorks() []string {
	return []string{
		`site:tgstat.com affiliate media buying`,
		`site:tgstat.com igaming`,
		`site:telemetr.io affiliate`,
		`site:affiliatefix.com "telegram channel"`,
		`site:affiliatefix.com t.me affiliate`,
		`site:afflift.com telegram group`,
		`site:blackhatworld.com "telegram" affiliate chat`,
		`site:stmforum.com telegram channel`,
		`site:gpwa.org telegram affiliate`,
		`site:reddit.com/r/affiliatemarketing telegram group`,
		`"telegram channels" affiliate marketing list`,
		`"telegram groups" media buying tracker`,
		`best telegram chats affiliate igaming`,
	}
}

func serpHarvestTGCatalogDorks(icpDorks []string) []string {
	if len(icpDorks) == 0 {
		return fallbackTGCatalogMetaDorks()
	}
	out := make([]string, 0, len(icpDorks))
	for _, dork := range icpDorks {
		dork = strings.TrimSpace(dork)
		if dork != "" {
			out = append(out, dork)
		}
	}
	return out
}

// HarvestTGCatalogSources finds web pages that list Telegram channels (meta-discovery).
func (c *Crawler) HarvestTGCatalogSources(ctx context.Context) error {
	icpPath := discover.ResolveICPPath("")
	icp, err := discover.LoadICP(icpPath)
	dorks := fallbackTGCatalogMetaDorks()
	if err == nil && len(icp.TGCatalogDorks) > 0 {
		dorks = serpHarvestTGCatalogDorks(icp.TGCatalogDorks)
	}
	dorks = dorkdisable.FilterActiveDorks(c.disabledDorksPath, dorks)
	dorks = selectSerpDorks(dorks, c.dorkOffset, c.dorkBatch, c.telegramDorkMax)

	catalogPath := defaultTGCatalogPagesPath
	var added int
	for _, dork := range dorks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		results, err := c.searchDork(ctx, dork)
		if err != nil {
			slog.Warn("tg catalog meta serp failed", "dork", dork, "error", err)
			continue
		}
		urls := make([]string, 0, len(results))
		for _, res := range results {
			if res.URL != "" {
				urls = append(urls, res.URL)
			}
		}
		n, err := appendTGCatalogPageDiscoveries(catalogPath, dork, urls)
		if err != nil {
			slog.Warn("tg catalog page registry write failed", "error", err)
			continue
		}
		added += n
	}
	slog.Info("tg catalog source harvest finished", "new_pages", added)
	return nil
}

// CrawlTGCatalogPages fetches catalog pages and extracts t.me handles into the channel registry.
func (c *Crawler) CrawlTGCatalogPages(ctx context.Context, maxPages int) error {
	if maxPages <= 0 {
		maxPages = 25
	}
	catalogPath := defaultTGCatalogPagesPath
	due := listTGCatalogPagesDue(catalogPath, maxPages, 14)
	if len(due) == 0 {
		slog.Info("tg catalog page crawl finished", "pages", 0, "new_handles", 0)
		return nil
	}

	var totalHandles int
	for _, page := range due {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		body, err := c.fetchPageHTML(ctx, page.URL)
		if err != nil {
			slog.Warn("tg catalog page fetch failed", "url", page.URL, "error", err)
			_ = markTGCatalogPageCrawled(catalogPath, page.URL, 0, err.Error())
			continue
		}
		handles := len(extract.TelegramHandles(body))
		invites := len(extract.TelegramInviteHashes(body))
		results := []SERPResult{{
			URL:     page.URL,
			Title:   page.URL,
			Snippet: body,
			Domain:  catalogHost(strings.ToLower(page.URL)),
		}}
		query := "tg_catalog:" + page.URL
		if err := appendTelegramChannelDiscoveries(defaultTGChannelsPath, query, results); err != nil {
			slog.Warn("tg catalog channel registry write failed", "url", page.URL, "error", err)
			_ = markTGCatalogPageCrawled(catalogPath, page.URL, 0, err.Error())
			continue
		}
		totalHandles += handles + invites
		_ = markTGCatalogPageCrawled(catalogPath, page.URL, handles+invites, "")
		slog.Debug("tg catalog page crawled", "url", page.URL, "handles", handles, "invites", invites)
	}
	slog.Info("tg catalog page crawl finished", "pages", len(due), "handles_seen", totalHandles)
	return nil
}
