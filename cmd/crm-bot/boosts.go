package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/sink"
	"github.com/spf13/cobra"
)

func newBoostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boosts",
		Short: "Manage crm_boosts false-negative queue",
	}
	cmd.AddCommand(newBoostsListCmd(), newBoostsDismissCmd(), newBoostsPromoteCmd())
	return cmd
}

func newBoostsListCmd() *cobra.Command {
	var status string
	var limit int64
	c := &cobra.Command{
		Use:   "list",
		Short: "List crm_boosts rows",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				docs, err := s.ListAllBoosts(cmd.Context(), strings.TrimSpace(status), limit)
				if err != nil {
					return err
				}
				if len(docs) == 0 {
					_, _ = fmt.Fprintln(out, "no boosts")
					return nil
				}
				for _, doc := range docs {
					id := ""
					if !doc.ID.IsZero() {
						id = doc.ID.Hex()
					}
					_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", id, doc.Status, doc.Source, truncateBoost(doc.Snippet, 80))
				}
				return nil
			})
		},
	}
	c.Flags().StringVar(&status, "status", sink.CrmBoostPending, "filter by status")
	c.Flags().Int64Var(&limit, "limit", 50, "max rows")
	return c
}

func newBoostsDismissCmd() *cobra.Command {
	var id, why string
	c := &cobra.Command{
		Use:   "dismiss",
		Short: "Dismiss a pending boost",
		RunE: func(cmd *cobra.Command, args []string) error {
			id = strings.TrimSpace(id)
			if id == "" {
				return fmt.Errorf("--id required")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				if err := s.ResolveBoost(cmd.Context(), id, sink.CrmBoostDismissed, "", why); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "dismissed boost %s\n", id)
				return nil
			})
		},
	}
	c.Flags().StringVar(&id, "id", "", "boost Mongo _id")
	c.Flags().StringVar(&why, "why", "manual dismiss", "outcome note")
	return c
}

func newBoostsPromoteCmd() *cobra.Command {
	var id, hashID, why string
	c := &cobra.Command{
		Use:   "promote",
		Short: "Mark boost promoted (optional linked lead hash)",
		RunE: func(cmd *cobra.Command, args []string) error {
			id = strings.TrimSpace(id)
			if id == "" {
				return fmt.Errorf("--id required")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(_ context.Context, out io.Writer, s *store.LeadStore) error {
				if err := s.ResolveBoost(cmd.Context(), id, sink.CrmBoostPromoted, strings.TrimSpace(hashID), why); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "promoted boost %s\n", id)
				return nil
			})
		},
	}
	c.Flags().StringVar(&id, "id", "", "boost Mongo _id")
	c.Flags().StringVar(&hashID, "hash", "", "optional lead hash_id")
	c.Flags().StringVar(&why, "why", "manual promote", "outcome note")
	return c
}

func truncateBoost(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
