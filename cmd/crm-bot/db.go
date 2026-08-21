package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/bidshard/parser/internal/crm/app"
	"github.com/bidshard/parser/internal/crm/config"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/spf13/cobra"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage Mongo lead database (on-server admin)",
		Long: `Database maintenance for CRM leads, notes, and tags.

Use on the VPS where Mongo is reachable. For remote access from a laptop, use crm-bot api.

Destructive commands require --yes.

Examples:
  crm-bot db stats
  crm-bot db delete --hash abcdef12 --yes
  crm-bot db purge --status new --score-max 30 --yes
  crm-bot db set-status --hash abcdef12 --status spam
  crm-bot db entity-split --id ENTITY_ID --hash HASH --yes`,
	}

	cmd.AddCommand(newDBStatsCmd(), newDBDeleteCmd(), newDBPurgeCmd(), newDBSetStatusCmd(), newDBEntitySplitCmd())
	return cmd
}

func newDBStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Print lead counts by status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				stats, err := s.DBStats(cmd.Context())
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "leads total: %d\n", stats.TotalLeads)
				for _, row := range stats.ByStatus {
					status := row.Status
					if status == "" {
						status = "(empty)"
					}
					_, _ = fmt.Fprintf(out, "  %s: %d\n", status, row.Count)
				}
				return nil
			})
		},
	}
}

func newDBDeleteCmd() *cobra.Command {
	var hashID string
	var yes bool
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete one lead and its CRM notes/tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			hashID = strings.TrimSpace(hashID)
			if hashID == "" {
				return fmt.Errorf("--hash required")
			}
			if !yes {
				return fmt.Errorf("pass --yes to confirm delete")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				result, err := s.DeleteLeads(cmd.Context(), store.DeleteFilter{HashID: hashID})
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "deleted leads=%d notes=%d meta=%d\n", result.Leads, result.Notes, result.Meta)
				return nil
			})
		},
	}
	c.Flags().StringVar(&hashID, "hash", "", "lead hash_id or unique prefix")
	c.Flags().BoolVar(&yes, "yes", false, "confirm delete")
	return c
}

func newDBPurgeCmd() *cobra.Command {
	var (
		all          bool
		status       string
		sourcePrefix string
		scoreMax     int
		yes          bool
	)
	c := &cobra.Command{
		Use:   "purge",
		Short: "Delete leads matching filters (max 10000 per run)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("pass --yes to confirm purge")
			}
			filter := store.DeleteFilter{
				All:          all,
				Status:       status,
				SourcePrefix: sourcePrefix,
				ScoreMax:     scoreMax,
			}
			if err := filter.Validate(); err != nil {
				return err
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				result, err := s.DeleteLeads(cmd.Context(), filter)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "purged leads=%d notes=%d meta=%d\n", result.Leads, result.Notes, result.Meta)
				return nil
			})
		},
	}
	c.Flags().BoolVar(&all, "all", false, "delete all leads")
	c.Flags().StringVar(&status, "status", "", "lead status filter")
	c.Flags().StringVar(&sourcePrefix, "source-prefix", "", "source prefix filter (e.g. tgweb:)")
	c.Flags().IntVar(&scoreMax, "score-max", 0, "delete leads with score <= N")
	c.Flags().BoolVar(&yes, "yes", false, "confirm purge")
	return c
}

func newDBSetStatusCmd() *cobra.Command {
	var hashID, status string
	c := &cobra.Command{
		Use:   "set-status",
		Short: "Update lead status by hash",
		RunE: func(cmd *cobra.Command, args []string) error {
			hashID = strings.TrimSpace(hashID)
			if hashID == "" {
				return fmt.Errorf("--hash required")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				resolved, err := s.ResolveHashID(cmd.Context(), hashID)
				if err != nil {
					return err
				}
				if err := s.UpdateStatus(cmd.Context(), resolved, status); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "updated hash_id=%s status=%s\n", resolved, strings.TrimSpace(status))
				return nil
			})
		},
	}
	c.Flags().StringVar(&hashID, "hash", "", "lead hash_id or unique prefix")
	c.Flags().StringVar(&status, "status", "", "new|contacted|won|lost|spam|archived")
	return c
}

func newDBEntitySplitCmd() *cobra.Command {
	var entityID, hashID string
	var yes bool
	c := &cobra.Command{
		Use:   "entity-split",
		Short: "Detach one lead hash into a new entity graph node",
		Long: `Ops-only repair for false merges (e.g. shared domain, different buyers).

Moves hash_id from --id entity into a new entity derived from that lead's identity keys.
Requires --yes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID = strings.TrimSpace(entityID)
			hashID = strings.TrimSpace(hashID)
			if entityID == "" {
				return fmt.Errorf("--id required")
			}
			if hashID == "" {
				return fmt.Errorf("--hash required")
			}
			if !yes {
				return fmt.Errorf("pass --yes to confirm entity split")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				if !s.EntitiesEnabled() {
					return fmt.Errorf("entity collection not configured (ENTITY_COLLECTION)")
				}
				result, err := s.SplitEntityHash(cmd.Context(), entityID, hashID)
				if err != nil {
					return err
				}
				if result.SourceDeleted {
					_, _ = fmt.Fprintf(out, "split hash_id=%s -> new entity_id=%s (source %s deleted, was last hash)\n",
						result.HashID, result.NewEntityID, result.SourceEntityID)
					return nil
				}
				_, _ = fmt.Fprintf(out, "split hash_id=%s from entity_id=%s -> new entity_id=%s\n",
					result.HashID, result.SourceEntityID, result.NewEntityID)
				return nil
			})
		},
	}
	c.Flags().StringVar(&entityID, "id", "", "source entity_id")
	c.Flags().StringVar(&hashID, "hash", "", "lead hash_id or unique prefix")
	c.Flags().BoolVar(&yes, "yes", false, "confirm split")
	return c
}

func withLeadStore(ctx context.Context, out io.Writer, fn func(context.Context, io.Writer, *store.LeadStore) error) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	if strings.TrimSpace(cfg.MongoURI) == "" {
		return fmt.Errorf("MONGO_URI empty")
	}
	leadStore, client, err := app.OpenLeadStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.CloseMongo(client)
	return fn(ctx, out, leadStore)
}
