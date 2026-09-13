package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/bidshard/parser/internal/crm/app"
	"github.com/bidshard/parser/internal/crm/config"
	"github.com/bidshard/parser/internal/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crm-bot",
	Short: "BidShard CRM sidecar for lead-intent-processor",
	Long: `HTTP sidecar for CRM leads in Mongo + Telegram export/notify bot.

Quick start (server):
  crm-bot config check
  crm-bot run

Parser webhook: POST /v1/leads with Bearer CRM_WEBHOOK_SECRET.
Ops: Telegram bot (/stats, /list, /export) and SSH + Mongo on VPS.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q - run 'crm-bot help' for usage", args[0])
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(
		newRunCmd(),
		newConfigCmd(),
		newVersionCmd(),
		newSyncOSINTCmd(),
	)
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the CRM HTTP server until interrupted",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if errs := cfg.ValidateForRun(); len(errs) > 0 {
				return fmt.Errorf("config invalid: %s", errs[0])
			}

			log.Init(cfg.LogFormat, cfg.LogLevel)

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			if err := app.Run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		},
	}
}
