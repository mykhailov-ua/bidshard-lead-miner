package cmd

import "github.com/spf13/cobra"

func newTelegramCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "Run the Telethon sidecar and ingest stdout",
		Long: `Start the Python Telethon scraper and pipe NDJSON into the parser.

Requires TELEGRAM_API_ID, TELEGRAM_API_HASH, and a session file.
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
	return cmd
}
