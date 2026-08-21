package main

import "github.com/spf13/cobra"

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the polling scan loop until interrupted",
		Long:  "Poll sources on PARSER_POLL_SEC interval (default 120s). Use 'parser scan' for a single round.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return globalOpts.run(cmd.Context())
		},
	}
}
