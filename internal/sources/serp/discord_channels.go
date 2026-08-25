package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sources/discord"
)

const defaultDiscordRegistryPath = "data/runtime/discovered_discord_invites.json"

// HarvestDiscordInvites appends public invite codes to the registry (no bot join).
func (c *Crawler) HarvestDiscordInvites(ctx context.Context, registryPath string) error {
	if registryPath == "" {
		registryPath = defaultDiscordRegistryPath
	}
	dorks := []string{
		`site:discord.gg "voluum"`,
		`site:discord.gg "affiliate" tracker`,
		`"discord.gg" igaming affiliate`,
	}
	icpPath := discover.ResolveICPPath("")
	if icp, err := discover.LoadICP(icpPath); err == nil {
		// Reuse discord-tagged SERP dorks from discover.icp.json when present.
		for _, d := range icp.SerpDorks {
			if strings.Contains(strings.ToLower(d), "discord") {
				dorks = append(dorks, d)
			}
		}
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
			slog.Warn("discord discover serp failed", "dork", dork, "error", err)
			continue
		}
		hints := map[string]string{}
		var codes []string
		for _, res := range results {
			for _, code := range discord.ExtractInviteCodes(res.URL + " " + res.Title + " " + res.Snippet) {
				hints[code] = strings.TrimSpace(res.Title + " " + res.Snippet)
				codes = append(codes, code)
			}
		}
		n, err := discord.AppendInvites(registryPath, "serp", dork, codes, hints)
		if err != nil {
			slog.Warn("discord invite registry write failed", "error", err)
			continue
		}
		added += n
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("discord", added)
	}
	slog.Info("discord invite discover finished", "new_invites", added)
	return nil
}
