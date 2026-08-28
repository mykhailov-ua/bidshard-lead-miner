package serp

import (
	"context"
	"log/slog"

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
	dorks := icp.SerpDorks
	if len(dorks) == 0 {
		dorks = fallbackTelegramCatalogDorks()
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

func fallbackTelegramCatalogDorks() []string {
	return []string{
		`site:t.me voluum alternative`,
		`site:t.me/+ affiliate igaming`,
		`site:t.me/joinchat affiliate`,
		`site:affiliatefix.com t.me tracker`,
		`site:blackhatworld.com t.me affiliate`,
	}
}
