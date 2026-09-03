package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/bgworker"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/domaincascade"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/metrics"
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
				return serp.RunBGHarvest(ctx, cfg, "serp_telegram_catalog", func(ctx context.Context) error {
					return serp.NewCrawler(cfg, nil).HarvestTelegramCatalog(ctx)
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
		{
			Name:          "domain_cascade_registry",
			Interval:      cfg.BGSourceRegistrySyncInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				_, _, err := domaincascade.SyncDiscoveryRegistries(domaincascade.Config{
					RegistryPath:        cfg.SourceRegistryPath,
					TelegramDomainsPath: cfg.TelegramDomainsPath,
					ForumRegistryPath:   cfg.ForumRegistryPath,
					WebPainRegistryPath: cfg.WebPainRegistryPath,
				})
				return err
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
				InitialDelay:  15 * time.Minute,
				SkipIfRunning: true,
				Run: func(ctx context.Context) error {
					err := telethon.RunDiscover(ctx, telethon.Options{
						ConfigPath: cfg.TelegramConfigPath,
						PythonBin:  cfg.TelethonPython,
					})
					if err != nil {
						metrics.RecordTelethonSidecarFailed()
					}
					return err
				},
			},
			bgworker.Job{
				Name:          "telegram_scrape",
				Interval:      cfg.BGTelegramScrapeInterval,
				InitialDelay:  90 * time.Second,
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
						CursorDBPath: cfg.TelegramCursorDBPath,
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
				return serp.RunBGHarvest(ctx, cfg, "discord_invite_discover", func(ctx context.Context) error {
					return serp.NewCrawler(cfg, nil).HarvestDiscordInvites(ctx, cfg.DiscordRegistryPath)
				})
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

	if cfg.ParserDiscoverFeedback && deps != nil && deps.sourceStats != nil && cfg.BGDiscoverFeedbackInterval > 0 {
		feedDeps := deps
		jobs = append(jobs, bgworker.Job{
			Name:          "discover_feedback",
			Interval:      cfg.BGDiscoverFeedbackInterval,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				result, err := RunDiscoverFeedback(ctx, cfg, feedDeps.mongoClient, feedDeps.sourceStats, feedDeps.keywordStats, feedDeps.registry)
				if err != nil {
					return err
				}
				if result.PrunedDorks > 0 || result.OutcomeReportPath != "" {
					slog.Info("discover feedback finished",
						"outcome_report", result.OutcomeReportPath,
						"keyword_tune", result.KeywordTunePath,
						"keyword_tune_applied", result.KeywordTuneApplied,
						"sales_feedback_ru", result.SalesFeedbackPath,
						"pruned_dorks", result.PrunedDorks,
					)
				}
				return nil
			},
		})
	}

	bgworker.Run(ctx, wg, jobs)
}
