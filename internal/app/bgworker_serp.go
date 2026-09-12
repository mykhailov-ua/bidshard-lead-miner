package app

import (
	"context"

	"github.com/bidshard/parser/internal/bgworker"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/domaincascade"
	"github.com/bidshard/parser/internal/sources/serp"
)

// serpHarvestBGJobOrder documents periodic SERP harvest registration order.
// Jobboard must register before telegram catalog (buyer path before long sweep).
var serpHarvestBGJobOrder = []string{
	"serp_jobboard_urls",
	"serp_forum_threads",
	"serp_employer_reverse",
	"serp_telegram_catalog",
	"serp_web_pain_catalog",
}

func serpHarvestBackgroundJobs(cfg config.Config) []bgworker.Job {
	return []bgworker.Job{
		{
			Name:          "serp_jobboard_urls",
			Interval:      cfg.BGForumDiscoverInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.RunBGHarvest(ctx, cfg, "serp_jobboard_urls", func(ctx context.Context) error {
					crawler := serp.NewCrawler(cfg, nil)
					if err := crawler.HarvestJobboardURLs(ctx, cfg.JobboardRegistryPath); err != nil {
						return err
					}
					return crawler.HarvestEmployerReverse(ctx, serp.EmployerReverseConfigFrom(cfg))
				})
			},
		},
		{
			Name:          "serp_forum_threads",
			Interval:      cfg.BGForumDiscoverInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.RunBGHarvest(ctx, cfg, "serp_forum_threads", func(ctx context.Context) error {
					return serp.NewCrawler(cfg, nil).HarvestForumThreads(ctx, cfg.ForumRegistryPath)
				})
			},
		},
		{
			Name:          "serp_employer_reverse",
			Interval:      cfg.BGForumDiscoverInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.RunBGHarvest(ctx, cfg, "serp_employer_reverse", func(ctx context.Context) error {
					return serp.NewCrawler(cfg, nil).HarvestEmployerReverse(ctx, serp.EmployerReverseConfigFrom(cfg))
				})
			},
		},
		{
			Name:          "serp_telegram_catalog",
			Interval:      cfg.BGSerpTelegramInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.RunBGHarvest(ctx, cfg, "serp_telegram_catalog", func(ctx context.Context) error {
					return serp.NewCrawler(cfg, nil).HarvestTelegramCatalog(ctx)
				})
			},
		},
		{
			Name:          "serp_web_pain_catalog",
			Interval:      cfg.BGForumDiscoverInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.RunBGHarvest(ctx, cfg, "serp_web_pain_catalog", func(ctx context.Context) error {
					return serp.NewCrawler(cfg, nil).HarvestWebPainCatalog(ctx, cfg.WebPainRegistryPath, domaincascade.Config{
						RegistryPath:        cfg.SourceRegistryPath,
						TelegramDomainsPath: cfg.TelegramDomainsPath,
					})
				})
			},
		},
	}
}
