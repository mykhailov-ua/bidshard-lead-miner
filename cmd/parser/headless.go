package main

import (
	"fmt"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/log"
	"github.com/spf13/cobra"
)

func newHeadlessCmd() *cobra.Command {
	var limit int
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "headless",
		Short: "Deferred Playwright headless queue (nightly drain)",
	}
	drainCmd := &cobra.Command{
		Use:   "drain",
		Short: "Render queued URLs with Playwright and emit leads",
		Long: `Process PARSER_LANDER_HEADLESS_DEFER queue (static crawl failures).

Run from cron once per night with docker-compose.headless.yaml profile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			if limit > 0 {
				cfg.LanderHeadlessDrainLimit = limit
			}
			cfg.LanderHeadless = true
			log.Init(globalOpts.effectiveLogFormat(cfg), globalOpts.effectiveLogLevel(cfg))
			if dryRun {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would drain up to %d items from %s\n", cfg.LanderHeadlessDrainLimit, cfg.LanderHeadlessQueuePath)
				return nil
			}
			return app.RunHeadlessDrainCLI(cmd.Context(), cfg)
		},
	}
	drainCmd.Flags().IntVar(&limit, "limit", 0, "max URLs to drain (default from PARSER_LANDER_HEADLESS_DRAIN_LIMIT)")
	drainCmd.Flags().BoolVar(&dryRun, "dry-run", false, "print queue path and limit only")
	cmd.AddCommand(drainCmd)
	return cmd
}
