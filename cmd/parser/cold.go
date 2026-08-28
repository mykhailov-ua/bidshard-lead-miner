package main

import (
	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/pretty"
	"github.com/spf13/cobra"
)

func newColdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cold",
		Short: "Cold-path Gemini utilities",
	}
	cmd.AddCommand(newColdReportCmd())
	return cmd
}

func newColdReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Run one junk report cycle (Gemini aggregate + optional diffs)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := app.RunColdReport(cmd.Context(), cfg); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			pretty.StatusOK(out, cliColor(out), "cold report complete")
			return nil
		},
	}
}
