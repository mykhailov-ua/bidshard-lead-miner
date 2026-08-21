package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/crm/config"
	"github.com/bidshard/parser/internal/sink"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate CRM bot configuration",
	}

	var showFormat string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Print effective configuration (secrets masked)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(cmd.Context(), cmd.OutOrStdout(), showFormat)
		},
	}
	showCmd.Flags().StringVar(&showFormat, "format", "text", "output format: text|json")
	cmd.AddCommand(showCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Validate CRM env and Mongo connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigCheck(cmd.Context(), cmd.OutOrStdout())
		},
	})

	return cmd
}

func runConfigShow(ctx context.Context, out io.Writer, format string) error {
	_ = ctx
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	view := cfg.View()
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(view)
	default:
		return view.WriteText(out)
	}
}

func runConfigCheck(ctx context.Context, out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(out, "error  config load: %v\n", err)
		return fmt.Errorf("config load: %w", err)
	}

	var errors []string
	var warnings []string

	errors = append(errors, cfg.ValidateForRun()...)

	if cfg.MongoURI != "" {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := pingMongo(pingCtx, cfg.MongoURI)
		cancel()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("MongoDB ping failed: %v", err))
		} else {
			_, _ = fmt.Fprintln(out, "ok  MongoDB reachable")
		}
	}

	if strings.TrimSpace(cfg.WebhookSecret) == "" {
		warnings = append(warnings, "CRM_WEBHOOK_SECRET empty - inbound webhook accepts unauthenticated POST")
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintf(out, "warn  %s\n", w)
	}
	for _, e := range errors {
		_, _ = fmt.Fprintf(out, "error  %s\n", e)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d configuration error(s)", len(errors))
	}
	_, _ = fmt.Fprintln(out, "ok  crm-bot config check passed")
	return nil
}

func pingMongo(ctx context.Context, uri string) error {
	client, err := sink.ConnectMongoClient(ctx, uri)
	if err != nil {
		return err
	}
	discCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return client.Disconnect(discCtx)
}
