package main

import (
	"context"

	"github.com/bidshard/parser/internal/config"
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
	return cmd
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
