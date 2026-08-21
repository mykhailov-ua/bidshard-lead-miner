package serp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/httpclient"
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

func (c *Crawler) searchDork(ctx context.Context, dork string) ([]SERPResult, error) {
	params := url.Values{}
	params.Set("q", dork)
	reqURL := c.baseURL
	if !strings.HasSuffix(reqURL, "/") {
		reqURL += "/"
	}
	if !strings.Contains(reqURL, "?") {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	body, status, err := httpclient.DoBytes(c.client, req, 2<<20)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("http %d", status)
	}
	return parseSERPResults(string(body)), nil
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
