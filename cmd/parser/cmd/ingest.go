package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newIngestCmd() *cobra.Command {
	var fixture string
	var fromStdin bool

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest NDJSON tasks from a file or stdin",
		Long: `Process NDJSON fixture tasks through the pipeline.

Examples:
  parser ingest --fixture=testdata/sample.ndjson
  cat tasks.ndjson | parser ingest`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalOpts
			opts.scanOnce = true
			if fixture != "" {
				opts.fixture = fixture
			} else if fromStdin || !isTerminal() {
				opts.ingestStdin = true
			} else {
				return cmd.Help()
			}
			return opts.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&fixture, "fixture", "", "NDJSON fixture file path")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read NDJSON from stdin")
	return cmd
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
