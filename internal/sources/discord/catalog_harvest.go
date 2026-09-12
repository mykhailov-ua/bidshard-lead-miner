package discord

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/bidshard/parser/internal/httpclient"
)

var catalogQueries = []string{
	"voluum affiliate",
	"keitaro tracker",
	"binom affiliate",
	"igaming media buying",
	"affiliate marketing",
	"media buying arbitrage",
	"redtrack",
	"self hosted tracker",
}

// HarvestCatalogInvites scrapes public listing sites (disboard) without SERP.
func HarvestCatalogInvites(ctx context.Context, registryPath string, client *http.Client) (int, error) {
	if client == nil {
		client = httpclient.Shared(20 * time.Second)
	}
	var added int
	for _, q := range catalogQueries {
		select {
		case <-ctx.Done():
			return added, ctx.Err()
		default:
		}
		n, err := harvestDisboardSearch(ctx, client, registryPath, q)
		if err != nil {
			slog.Warn("discord disboard harvest failed", "query", q, "error", err)
			continue
		}
		added += n
	}
	slog.Info("discord catalog harvest finished", "new_invites", added)
	return added, nil
}

func harvestDisboardSearch(ctx context.Context, client *http.Client, registryPath, query string) (int, error) {
	u := "https://disboard.org/search?keyword=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BidShardParser/1.0)")
	body, status, err := httpclient.DoBytes(client, req, 4<<20)
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("disboard http %d", status)
	}
	text := string(body)
	codes := ExtractInviteCodes(text)
	if len(codes) == 0 {
		return 0, nil
	}
	hints := map[string]string{}
	for _, code := range codes {
		hints[code] = query
	}
	return AppendInvites(registryPath, "disboard", query, codes, hints)
}
