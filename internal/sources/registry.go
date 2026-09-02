package sources

import (
	"context"
	"strings"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sourcedisable"
	"github.com/bidshard/parser/internal/sources/ct"
	"github.com/bidshard/parser/internal/sources/discord"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/github"
	"github.com/bidshard/parser/internal/sources/lander"
	"github.com/bidshard/parser/internal/sources/reddit"
	"github.com/bidshard/parser/internal/sources/reviews"
	"github.com/bidshard/parser/internal/sources/serp"
	"github.com/bidshard/parser/internal/sources/supply"
	"github.com/bidshard/parser/internal/sources/webpain"
)

func Build(cfg config.Config) []Source {
	names := parseSourceList(cfg.Source)
	if len(names) == 0 {
		return nil
	}

	var out []Source
	for _, name := range names {
		if sourcedisable.IsDisabled(cfg.DisabledSourcesPath, name) {
			continue
		}
		if src, ok := buildOne(cfg, name); ok {
			out = append(out, src)
		}
	}
	if cfg.ParserSourcePriority {
		OrderByCollectPriority(out)
	}
	return out
}

// ParseSourceNames splits PARSER_SOURCE / --source into canonical source names.
func ParseSourceNames(raw string) []string {
	return parseSourceList(raw)
}

func parseSourceList(raw string) []string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil
	}
	if raw == "all" {
		// Precision default: omit lander (competitor HTML junk). Opt in: PARSER_SOURCE=...,lander
		return []string{"forum", "supply", "reddit", "discord", "serp"}
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = canonicalSourceName(p)
		if p == "" {
			continue
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func canonicalSourceName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "warrior" {
		return "forum"
	}
	return name
}

func buildOne(cfg config.Config, name string) (Source, bool) {
	switch name {
	case "supply":
		return wrapSupply(supply.NewCrawler(cfg, nil)), true
	case "forum":
		return wrapForum(forum.NewAdapter(cfg, nil)), true
	case "lander":
		var headless lander.HeadlessFetcher = lander.DisabledHeadless{}
		if cfg.LanderHeadless {
			headless = lander.NewPlaywrightPoolFetcher(2, cfg.HTTPTimeout)
		}
		return wrapLander(lander.NewCrawler(cfg, nil, headless)), true
	case "reddit":
		return wrapReddit(reddit.NewCrawler(cfg)), true
	case "discord":
		return wrapDiscord(discord.NewCrawler(cfg)), true
	case "ct":
		return wrapCT(ct.NewCrawler(cfg, nil)), true
	case "reviews":
		return wrapReviews(reviews.NewCrawler(cfg, nil)), true
	case "github":
		if !cfg.ParserGitHubEnabled {
			return nil, false
		}
		return wrapGitHub(github.NewCrawler(cfg)), true
	case "serp":
		return wrapSERP(serp.NewCrawler(cfg, nil)), true
	case "webpain":
		return wrapWebPain(webpain.NewAdapter(cfg, nil)), true
	default:
		return nil, false
	}
}

type serpSource struct {
	inner *serp.Crawler
}

func wrapSERP(inner *serp.Crawler) Source {
	return &serpSource{inner: inner}
}

func (s *serpSource) Name() string {
	return s.inner.Name()
}

func (s *serpSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type gitHubSource struct {
	inner *github.Crawler
}

func wrapGitHub(inner *github.Crawler) Source {
	return &gitHubSource{inner: inner}
}

func (s *gitHubSource) Name() string {
	return s.inner.Name()
}

func (s *gitHubSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type reviewsSource struct {
	inner *reviews.Crawler
}

func wrapReviews(inner *reviews.Crawler) Source {
	return &reviewsSource{inner: inner}
}

func (s *reviewsSource) Name() string {
	return s.inner.Name()
}

func (s *reviewsSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type ctSource struct {
	inner *ct.Crawler
}

func wrapCT(inner *ct.Crawler) Source {
	return &ctSource{inner: inner}
}

func (s *ctSource) Name() string {
	return s.inner.Name()
}

func (s *ctSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type supplySource struct {
	inner *supply.Crawler
}

func wrapSupply(inner *supply.Crawler) Source {
	return &supplySource{inner: inner}
}

func (s *supplySource) Name() string {
	return s.inner.Name()
}

func (s *supplySource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type forumSource struct {
	inner *forum.Adapter
}

func wrapForum(inner *forum.Adapter) Source {
	return &forumSource{inner: inner}
}

func (s *forumSource) Name() string {
	return s.inner.Name()
}

func (s *forumSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type webPainSource struct {
	inner *webpain.Adapter
}

func wrapWebPain(inner *webpain.Adapter) Source {
	return &webPainSource{inner: inner}
}

func (s *webPainSource) Name() string {
	return s.inner.Name()
}

func (s *webPainSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type landerSource struct {
	inner *lander.Crawler
}

func wrapLander(inner *lander.Crawler) Source {
	return &landerSource{inner: inner}
}

func (s *landerSource) Name() string {
	return s.inner.Name()
}

func (s *landerSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type redditSource struct {
	inner *reddit.Crawler
}

func wrapReddit(inner *reddit.Crawler) Source {
	return &redditSource{inner: inner}
}

func (s *redditSource) Name() string {
	return s.inner.Name()
}

func (s *redditSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type discordSource struct {
	inner *discord.Crawler
}

func wrapDiscord(inner *discord.Crawler) Source {
	return &discordSource{inner: inner}
}

func (s *discordSource) Name() string {
	return s.inner.Name()
}

func (s *discordSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}
