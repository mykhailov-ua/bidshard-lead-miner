package main

import (
	"fmt"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/spf13/cobra"
)

func newAutoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Automation health and weekly reports",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Print discover registry sizes, queues, proxy burn, defer backlog",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			st := app.CollectAutoStatus(cmd.Context(), cfg)
			app.WriteAutoStatus(cmd.OutOrStdout(), st)
			return nil
		},
	})

	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Append automation snapshot JSONL (no PII)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			st := app.CollectAutoStatus(cmd.Context(), cfg)
			if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
				app.WriteAutoStatus(cmd.OutOrStdout(), st)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "dry-run: report not written")
				return nil
			}
			path := cfg.AutoReportPath
			if err := app.WriteAutoReportJSONL(path, st); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok  wrote %s\n", path)
			return nil
		},
	}
	reportCmd.Flags().Bool("dry-run", false, "print status only, do not write JSONL")
	cmd.AddCommand(reportCmd)

	return cmd
}
