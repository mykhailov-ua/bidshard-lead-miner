package main

import (
	"context"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/sources/serp"
	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "One-shot discovery jobs (SERP harvest from discover.icp.json)",
	}
	cmd.AddCommand(newDiscoverSerpCmd())
	cmd.AddCommand(newDiscoverJobboardCmd())
	cmd.AddCommand(newDiscoverTGCatalogCmd())
	cmd.AddCommand(newDiscoverInfraClustersCmd())
	return cmd
}

func newDiscoverInfraClustersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "infra-clusters",
		Short: "Score data/runtime/infra_clusters.json for H6 manual outreach triage",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			file, err := discover.LoadInfraClusters(cfg.InfraClustersPath)
			if err != nil {
				return err
			}
			for _, cluster := range file.Clusters {
				cmd.Printf(
					"ip=%s score=%d domains=%d country=%s tracker=%s sample=%s\n",
					cluster.IP,
					cluster.Score,
					len(cluster.Domains),
					cluster.Country,
					cluster.TrackerHint,
					cluster.SampleDomain,
				)
			}
			return nil
		},
	}
}

func newDiscoverTGCatalogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tg-catalog",
		Short: "Harvest TG channel catalog pages (SERP) and extract t.me handles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			crawler := serp.NewCrawler(cfg, nil)
			if err := crawler.HarvestTGCatalogSources(ctx); err != nil {
				return err
			}
			return crawler.CrawlTGCatalogPages(ctx, cfg.SerpTGCatalogCrawlMax)
		},
	}
}

func newDiscoverSerpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serp",
		Short: "Harvest job boards, forums, employers, and telegram channels from discover.icp.json dorks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return runDiscoverSerpHarvest(ctx, cfg, serp.NewCrawler(cfg, nil))
		},
	}
}

// discoverSerpHarvester is the SERP harvest sequence for parser discover serp.
type discoverSerpHarvester interface {
	HarvestJobboardURLs(ctx context.Context, registryPath string) error
	HarvestForumThreads(ctx context.Context, registryPath string) error
	HarvestEmployerReverse(ctx context.Context, cfg serp.EmployerReverseConfig) error
	HarvestTGCatalogSources(ctx context.Context) error
	CrawlTGCatalogPages(ctx context.Context, maxPages int) error
	HarvestTelegramCatalog(ctx context.Context) error
}

// runDiscoverSerpHarvest runs buyer-path discovery before the long telegram sweep.
func runDiscoverSerpHarvest(ctx context.Context, cfg config.Config, h discoverSerpHarvester) error {
	if err := h.HarvestJobboardURLs(ctx, cfg.JobboardRegistryPath); err != nil {
		return err
	}
	if err := h.HarvestForumThreads(ctx, cfg.ForumRegistryPath); err != nil {
		return err
	}
	if err := h.HarvestEmployerReverse(ctx, employerReverseConfig(cfg)); err != nil {
		return err
	}
	if err := h.HarvestTGCatalogSources(ctx); err != nil {
		return err
	}
	if err := h.CrawlTGCatalogPages(ctx, cfg.SerpTGCatalogCrawlMax); err != nil {
		return err
	}
	return h.HarvestTelegramCatalog(ctx)
}

func newDiscoverJobboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jobboard",
		Short: "Harvest DOU/Djinni job URLs and run employer reverse SERP (buyer-team fast path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			crawler := serp.NewCrawler(cfg, nil)
			if err := crawler.HarvestJobboardURLs(ctx, cfg.JobboardRegistryPath); err != nil {
				return err
			}
			return crawler.HarvestEmployerReverse(ctx, employerReverseConfig(cfg))
		},
	}
}

func employerReverseConfig(cfg config.Config) serp.EmployerReverseConfig {
	return serp.EmployerReverseConfigFrom(cfg)
}
