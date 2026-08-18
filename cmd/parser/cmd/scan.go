package cmd

import "github.com/spf13/cobra"

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Run a single scan round and exit",
		Long: `Run one scan round and exit. Default source is stub (no network).

Examples:
  parser scan
  parser scan --source=forum
  parser scan --source=stub --output=table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalOpts
			opts.scanOnce = true
			return opts.run(cmd.Context())
		},
	}
}
