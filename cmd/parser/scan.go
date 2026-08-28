package main

import "github.com/spf13/cobra"

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Run a single scan round and exit",
		Long: `Run one scan round and exit. Default sources are forum+supply+lander+reddit+discord+serp.

Examples:
  parser scan
  parser scan -s forum,reddit
  parser scan -o pretty
  parser scan --json > leads.jsonl
  parser scan --export=data/export/leads.jsonl -q`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalOpts
			opts.scanOnce = true
			return opts.run(cmd.Context())
		},
	}
}
