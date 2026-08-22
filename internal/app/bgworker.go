package app

import (
	"context"
	"sync"

	"github.com/bidshard/parser/internal/bgworker"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sources/serp"
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

	bgworker.Run(ctx, wg, jobs)
}
