package main

import (
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/telethon"
	"github.com/spf13/cobra"
)

func newTelegramCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "Run the Telethon sidecar and ingest stdout",
		Long: `Start the Python Telethon scraper and pipe NDJSON into the parser.

Requires TELEGRAM_API_ID, TELEGRAM_API_HASH, and an authorized session file.
First-time setup: parser telegram login --qr (recommended) or login with TELEGRAM_PHONE + TELEGRAM_CODE.
Use --dry-run to test without an MTProto session.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalOpts
			opts.telegramSidecar = true
			opts.scanOnce = true
			opts.telegramDryRun = dryRun
			return opts.run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Telethon dry-run (no MTProto session)")
	cmd.AddCommand(newTelegramLoginCmd())
	cmd.AddCommand(newTelegramDiscoverCmd())
	cmd.AddCommand(newTelegramExportRegistryCmd())
	cmd.AddCommand(newTelegramWebCmd())
	cmd.AddCommand(newTelegramDomainsCmd())
	cmd.AddCommand(newTelegramRealtimeCmd())
	return cmd
}

func newTelegramRealtimeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "realtime",
		Short: "Run Telethon NewMessage listener and ingest NDJSON",
		Long: `Long-running MTProto listener for configured Telegram channels.

Requires TELEGRAM_API_ID, TELEGRAM_API_HASH, authorized session, and TELEGRAM_REALTIME=1.
Shares session lock with scrape/discover; do not run both against the same session.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalOpts
			opts.telegramRealtime = true
			return opts.run(cmd.Context())
		},
	}
}

func newTelegramDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Maintain discovered Telegram domain registry",
	}
	cmd.AddCommand(newTelegramDomainsPruneCmd())
	cmd.AddCommand(newTelegramDomainsResetCmd())
	return cmd
}

func newTelegramDomainsResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset [domain ...]",
		Short: "Clear crawled_at for domains (re-crawl on next telegram web run)",
		// Does not add domains; only clears crawled_at on entries already in the registry file.
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			return app.RunTelegramDomainsReset(cmd.Context(), cfg, args)
		},
	}
}

func newTelegramDomainsPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Remove invalid hosts from discovered_telegram_domains.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			return app.RunTelegramDomainsPrune(cmd.Context(), cfg)
		},
	}
}

func newTelegramWebCmd() *cobra.Command {
	var onlyDomains string
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Crawl affiliate network sites from Telegram domain registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			if onlyDomains != "" {
				// Filter registry to listed domains; ignore crawled_at but require registry membership.
				cfg.TelegramWebDomains = splitCSV(onlyDomains)
			}
			return app.RunTelegramWeb(cmd.Context(), cfg)
		},
	}
	cmd.Flags().StringVar(&onlyDomains, "domains", "", "comma-separated domains to crawl (must exist in registry; ignores crawled_at)")
	return cmd
}

func splitCSV(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func newTelegramDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover Telegram channels (search + SERP + cross-mention)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			return telethon.RunDiscover(cmd.Context(), telethon.Options{
				ConfigPath: cfg.TelegramConfigPath,
				PythonBin:  cfg.TelethonPython,
			})
		},
	}
}

func newTelegramExportRegistryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export-registry",
		Short: "Export discovered_telegram_channels.json from crawler.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			return telethon.RunExportRegistry(cmd.Context(), telethon.Options{
				ConfigPath: cfg.TelegramConfigPath,
				PythonBin:  cfg.TelethonPython,
			})
		},
	}
}

func newTelegramLoginCmd() *cobra.Command {
	var qrLogin bool
	var fresh bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Telethon MTProto login",
		Long: `Authorize a user session for the Telethon sidecar.

QR login (recommended if app code does not arrive - Telegram no longer sends SMS to third-party apps):
  docker compose run --rm -it parser telegram login --qr

Phone + code (code only in official Telegram app, chat "Telegram"):
  docker compose run --rm parser telegram login -e TELEGRAM_PHONE=+...   # step 1: request code
  docker compose run --rm parser telegram login -e TELEGRAM_PHONE=+... -e TELEGRAM_CODE=12345

Session path: config/sources.telegram.yaml (default data/runtime/telethon.session).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}
			if err := telethon.RunLogin(cmd.Context(), telethon.Options{
				ConfigPath: cfg.TelegramConfigPath,
				PythonBin:  cfg.TelethonPython,
				LoginQR:    qrLogin,
				LoginFresh: fresh,
			}); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Telegram session saved. Run: parser telegram")
			return nil
		},
	}

	cmd.Flags().BoolVar(&qrLogin, "qr", false, "QR login (scan in Telegram app; no SMS code)")
	cmd.Flags().BoolVar(&fresh, "fresh", false, "delete existing session before login")
	return cmd
}
