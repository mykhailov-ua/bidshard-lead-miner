package app

import (
	"context"
	"sync"

	"github.com/bidshard/parser/internal/bgworker"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sources/serp"
	"github.com/bidshard/parser/internal/sources/tgweb"
	"github.com/bidshard/parser/internal/telethon"
)

func startBackgroundWorkers(ctx context.Context, cfg config.Config, deps *runtimeDeps, wg *sync.WaitGroup) {
	// Disabled for scan-once, telegram sidecar, or when PARSER_BG_WORKER is false.
	if !cfg.BGWorkerEnabled || cfg.ScanOnce || cfg.TelegramSidecar {
		return
	}

	jobs := []bgworker.Job{
		{
			Name:          "serp_telegram_catalog",
			Interval:      cfg.BGSerpTelegramInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.NewCrawler(cfg, nil).HarvestTelegramCatalog(ctx)
			},
		},
		{
			Name:          "serp_forum_threads",
			Interval:      cfg.BGForumDiscoverInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.NewCrawler(cfg, nil).HarvestForumThreads(ctx, cfg.ForumRegistryPath)
			},
		},
		{
			Name:          "source_registry_sync",
			Interval:      cfg.BGSourceRegistrySyncInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				_, err := tgweb.SyncToSourceRegistry(cfg.TelegramDomainsPath, cfg.SourceRegistryPath)
				return err
			},
		},
	}

	if cfg.BGTelegramEnabled && cfg.TelegramAPIHash != "" && cfg.TelegramAPIID > 0 {
		jobs = append(jobs,
			bgworker.Job{
				Name:          "telegram_discover",
				Interval:      cfg.BGTelegramDiscoverInterval,
				SkipIfRunning: true,
				Run: func(ctx context.Context) error {
					return telethon.RunDiscover(ctx, telethon.Options{
						ConfigPath: cfg.TelegramConfigPath,
						PythonBin:  cfg.TelethonPython,
					})
				},
			},
			bgworker.Job{
				Name:          "telegram_scrape",
				Interval:      cfg.BGTelegramScrapeInterval,
				SkipIfRunning: true,
				Run: func(ctx context.Context) error {
					return runTelegramSidecarOnce(ctx, cfg, deps)
				},
			},
			bgworker.Job{
				Name:          "telegram_web",
				Interval:      cfg.BGTelegramWebInterval,
				SkipIfRunning: true,
				Run: func(ctx context.Context) error {
					return runTelegramWebOnce(ctx, cfg, deps)
				},
			},
		)
	}

	if cfg.ParserChannelTriage && cfg.GeminiAPIKey != "" {
		if client, err := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel); err == nil {
			jobs = append(jobs, bgworker.Job{
				Name:          "telegram_channel_triage",
				Interval:      cfg.BGChannelTriageInterval,
				SkipIfRunning: true,
				Run: func(ctx context.Context) error {
					return telethon.RunChannelTriage(ctx, telethon.ChannelTriageConfig{
						ChannelsPath: cfg.TelegramChannelsPath,
					}, client)
				},
			})
		}
	}

	if cfg.ParserDomainTriage && cfg.GeminiAPIKey != "" {
		if client, err := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel); err == nil {
			jobs = append(jobs, bgworker.Job{
				Name:          "domain_triage",
				Interval:      cfg.BGDomainTriageInterval,
				SkipIfRunning: true,
				Run: func(ctx context.Context) error {
					return RunDomainTriage(ctx, DomainTriageConfig{
						RegistryPath: cfg.SourceRegistryPath,
						CachePath:    cfg.DomainTriageCachePath,
					}, client)
				},
			})
		}
	}

	if cfg.BGAutoReportInterval > 0 {
		jobs = append(jobs, bgworker.Job{
			Name:          "auto_report",
			Interval:      cfg.BGAutoReportInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				st := CollectAutoStatus(ctx, cfg)
				return WriteAutoReportJSONL(cfg.AutoReportPath, st)
			},
		})
	}

	if cfg.ParserAutoDiscover && cfg.BGDiscordDiscoverInterval > 0 {
		// Registry only; DISCORD_CHANNEL_IDS still required for crawl.
		jobs = append(jobs, bgworker.Job{
			Name:          "discord_invite_discover",
			Interval:      cfg.BGDiscordDiscoverInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return serp.NewCrawler(cfg, nil).HarvestDiscordInvites(ctx, cfg.DiscordRegistryPath)
			},
		})
	}

	if cfg.ParserSourceDisableGovernor && deps != nil && deps.sourceStats != nil && cfg.BGSourceDisableInterval > 0 {
		// Needs Mongo source_stats; sources.Build skips keys in disabled_sources.json.
		stats := deps.sourceStats
		jobs = append(jobs, bgworker.Job{
			Name:          "source_disable_governor",
			Interval:      cfg.BGSourceDisableInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				return RunSourceDisableGovernor(ctx, cfg, stats)
			},
		})
	}

	bgworker.Run(ctx, wg, jobs)
}
