package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/dorkdisable"
)

// HarvestTelegramCatalog searches the web for public and invite Telegram links only.
// Dorks come from config/discover.icp.json (lead ICP).
func (c *Crawler) HarvestTelegramCatalog(ctx context.Context) error {
	icpPath := discover.ResolveICPPath("")
	icp, err := discover.LoadICP(icpPath)
	if err != nil {
		slog.Warn("telegram catalog icp load failed, using embedded fallback", "path", icpPath, "error", err)
		icp.SerpDorks = fallbackTelegramCatalogDorks()
	}
	dorks := serpHarvestTelegramDorks(icp.SerpDorks)
	if len(dorks) == 0 {
		dorks = fallbackTelegramCatalogDorks()
	}
	dorks = dorkdisable.FilterActiveDorks(c.disabledDorksPath, dorks)
	dorks = limitSerpDorks(dorks, c.telegramDorkMax)

	var added int
	for _, dork := range dorks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		results, err := c.searchDork(ctx, dork)
		if err != nil {
			slog.Warn("telegram catalog serp failed", "dork", dork, "error", err)
			continue
		}
		before := len(readTGChannelFile(defaultTGChannelsPath).Channels)
		if err := appendTelegramChannelDiscoveries(defaultTGChannelsPath, dork, results); err != nil {
			slog.Warn("telegram catalog registry write failed", "error", err)
			continue
		}
		after := len(readTGChannelFile(defaultTGChannelsPath).Channels)
		added += after - before
	}
	slog.Info("telegram catalog harvest finished", "new_entries", added)
	return nil
}

// serpHarvestTelegramDorks returns ICP dorks for t.me channel harvest.
// Job-board dorks (DOU/Djinni) are excluded; they are handled by HarvestJobboardURLs.
func serpHarvestTelegramDorks(dorks []string) []string {
	all := serpHarvestDorksFromICP(dorks)
	out := make([]string, 0, len(all))
	for _, dork := range all {
		lower := strings.ToLower(dork)
		if strings.Contains(lower, "jobs.dou.ua") || strings.Contains(lower, "djinni.co") {
			continue
		}
		out = append(out, dork)
	}
	return out
}

func limitSerpDorks(dorks []string, max int) []string {
	if max <= 0 || len(dorks) <= max {
		return dorks
	}
	return dorks[:max]
}

func fallbackTelegramCatalogDorks() []string {
	return []string{
		`site:t.me voluum alternative`,
		`site:t.me/+ affiliate igaming`,
		`site:t.me/joinchat affiliate`,
		`site:affiliatefix.com t.me tracker`,
		`site:blackhatworld.com t.me affiliate`,
	}
}
