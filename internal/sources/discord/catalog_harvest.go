package discord

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/httpclient"
)

var disboardJoinRe = regexp.MustCompile(`/server/join/([a-zA-Z0-9-]{2,64})`)

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

const defaultSeedInvitesPath = "config/discord_seed_invites.txt"

// HarvestCatalogInvites scrapes public listing sites (disboard) without SERP.
func HarvestCatalogInvites(ctx context.Context, registryPath string, client *http.Client) (int, error) {
	if client == nil {
		client = httpclient.Shared(20 * time.Second)
	}
	added, err := HarvestSeedInvites(registryPath, os.Getenv("DISCORD_SEED_INVITES_PATH"))
	if err != nil {
		return added, err
	}
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

// HarvestSeedInvites loads curated public invite codes from config file.
func HarvestSeedInvites(registryPath, seedPath string) (int, error) {
	if seedPath == "" {
		seedPath = defaultSeedInvitesPath
	}
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var codes []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		code := strings.ToLower(line)
		if ValidInviteCode(code) {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return 0, nil
	}
	hints := map[string]string{}
	for _, code := range codes {
		hints[code] = "seed"
	}
	return AppendInvites(registryPath, "seed", "config", codes, hints)
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
	seen := map[string]struct{}{}
	var codes []string
	for _, code := range ExtractInviteCodes(text) {
		code = strings.ToLower(code)
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return 0, nil
	}
	hints := map[string]string{}
	for _, code := range codes {
		hints[code] = query
	}
	return AppendInvites(registryPath, "disboard", query, codes, hints)
}
