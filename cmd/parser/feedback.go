package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/spf13/cobra"
)

func newFeedbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feedback",
		Short: "Run discover outcome feedback loop (reports + dork prune)",
	}
	cmd.AddCommand(newFeedbackRunCmd())
	return cmd
}

func newFeedbackRunCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "run",
		Short: "Write outcome_dork_rank, keyword_tune, and disable weak dorks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			result, err := app.RunFeedbackCommand(ctx, cfg)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				raw, _ := json.MarshalIndent(result, "", "  ")
				_, _ = fmt.Fprintln(out, string(raw))
				return nil
			}
			printDiscoverFeedback(out, result)
			return nil
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON instead of colored summary")
	return c
}
