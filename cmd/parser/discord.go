package main

import (
	"fmt"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/discord"
	"github.com/bidshard/parser/internal/sources/serp"
	"github.com/spf13/cobra"
)

func newDiscordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discord",
		Short: "Discord invite/channel discovery",
	}
	cmd.AddCommand(newDiscordDiscoverCmd())
	return cmd
}

func newDiscordDiscoverCmd() *cobra.Command {
	fast := false
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Harvest public invites from catalogs and register readable channels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if _, err := discord.HarvestCatalogInvites(ctx, cfg.DiscordRegistryPath, nil); err != nil {
				return fmt.Errorf("discord catalog harvest: %w", err)
			}
			if !fast {
				crawler := serp.NewCrawler(cfg, nil)
				if err := crawler.HarvestDiscordInvites(ctx, cfg.DiscordRegistryPath); err != nil {
					return fmt.Errorf("discord serp harvest: %w", err)
				}
			}
			if err := discord.DiscoverChannels(ctx, cfg); err != nil {
				return fmt.Errorf("discord channel discover: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fast, "fast", false, "skip slow SERP harvest (disboard catalog only)")
	return cmd
}
