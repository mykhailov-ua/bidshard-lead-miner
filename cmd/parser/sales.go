package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/spf13/cobra"
)

func newSalesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sales",
		Short: "Russian JSON exports for sales managers",
	}
	cmd.AddCommand(newSalesExportCmd())
	return cmd
}

func newSalesExportCmd() *cobra.Command {
	var limit, minScore int
	var asJSON bool
	c := &cobra.Command{
		Use:   "export",
		Short: "Write leads_ru and junk_report_ru JSON under data/export/sales",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			result, err := app.RunSalesExport(ctx, cfg, limit, minScore)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				raw, _ := json.MarshalIndent(result, "", "  ")
				_, _ = fmt.Fprintln(out, string(raw))
				return nil
			}
			printSalesExport(out, result)
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 50, "max leads to export")
	c.Flags().IntVar(&minScore, "min-score", 0, "minimum intent score")
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON instead of colored summary")
	return c
}
