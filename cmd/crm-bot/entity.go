package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/bidshard/parser/internal/crm/apiclient"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/spf13/cobra"
)

func newEntityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entity",
		Short: "Entity-centric sales inbox (local Mongo)",
		Long: `Buyer graph admin: one entity card across forum/telegram/email sightings.

Examples:
  crm-bot entity inbox --min-tier warm --min-sightings 2
  crm-bot entity show --id ENTITY_ID
  crm-bot entity leads --id ENTITY_ID
  crm-bot entity suggestions
  crm-bot entity reclassify --id ENTITY_ID`,
	}
	cmd.AddCommand(
		newEntityInboxCmd(),
		newEntityShowCmd(),
		newEntityLeadsCmd(),
		newEntitySuggestionsCmd(),
		newEntityReclassifyCmd(),
	)
	return cmd
}

func newEntityInboxCmd() *cobra.Command {
	var (
		minTier      string
		minSightings int
		engageMin    int
		needsWork    bool
		limit        int64
		asJSON       bool
	)
	c := &cobra.Command{
		Use:   "inbox",
		Short: "List heat-ranked buyer entities for outreach",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(ctx context.Context, out io.Writer, s *store.LeadStore) error {
				if !s.EntitiesEnabled() {
					return fmt.Errorf("entity store not configured")
				}
				if limit <= 0 {
					limit = 20
				}
				if engageMin < 0 {
					engageMin = store.InboxEngagePriorityMin()
				}
				result, err := s.ListEntityInbox(ctx, store.EntityInboxFilter{
					MinTier:           strings.TrimSpace(minTier),
					MinSightings:      minSightings,
					OnlyNeedsWork:     needsWork,
					MinEngagePriority: engageMin,
					Limit:             limit,
				})
				if err != nil {
					return err
				}
				if asJSON {
					return writeJSON(out, result)
				}
				apiclient.WriteEntityInboxTable(out, result.Entities)
				return nil
			})
		},
	}
	c.Flags().StringVar(&minTier, "min-tier", "warm", "minimum heat tier: warm|hot|blazing")
	c.Flags().IntVar(&minSightings, "min-sightings", 2, "minimum sightings (cross-source story)")
	c.Flags().IntVar(&engageMin, "engage-min", -1, "minimum engage_priority (default from CRM_ENGAGE_PRIORITY_MIN)")
	c.Flags().BoolVar(&needsWork, "needs-work", false, "only needs_review, classify_force, or pending suggestions")
	c.Flags().Int64Var(&limit, "limit", 20, "max rows")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newEntityShowCmd() *cobra.Command {
	var entityID string
	var asJSON bool
	c := &cobra.Command{
		Use:   "show",
		Short: "Show one entity inbox card (SDR narrative)",
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID = strings.TrimSpace(entityID)
			if entityID == "" {
				return fmt.Errorf("--id required")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(ctx context.Context, out io.Writer, s *store.LeadStore) error {
				if !s.EntitiesEnabled() {
					return fmt.Errorf("entity store not configured")
				}
				card, err := s.GetEntityInbox(ctx, entityID)
				if err != nil {
					return err
				}
				if asJSON {
					return writeJSON(out, card)
				}
				apiclient.WriteEntityInboxCard(out, card)
				return nil
			})
		},
	}
	c.Flags().StringVar(&entityID, "id", "", "entity_id")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newEntityLeadsCmd() *cobra.Command {
	var entityID string
	var limit int64
	var asJSON bool
	c := &cobra.Command{
		Use:   "leads",
		Short: "List leads linked to an entity",
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID = strings.TrimSpace(entityID)
			if entityID == "" {
				return fmt.Errorf("--id required")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(ctx context.Context, out io.Writer, s *store.LeadStore) error {
				if !s.EntitiesEnabled() {
					return fmt.Errorf("entity store not configured")
				}
				if limit <= 0 {
					limit = 20
				}
				leads, err := s.ListEntityLeads(ctx, entityID, limit)
				if err != nil {
					return err
				}
				if asJSON {
					return writeJSON(out, map[string]any{"entity_id": entityID, "leads": leads})
				}
				apiclient.WriteLeadTable(out, leads)
				return nil
			})
		},
	}
	c.Flags().StringVar(&entityID, "id", "", "entity_id")
	c.Flags().Int64Var(&limit, "limit", 20, "max rows")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newEntitySuggestionsCmd() *cobra.Command {
	var limit int64
	var asJSON bool
	c := &cobra.Command{
		Use:   "suggestions",
		Short: "List pending entity merge/split suggestions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(ctx context.Context, out io.Writer, s *store.LeadStore) error {
				if !s.EntitiesEnabled() {
					return fmt.Errorf("entity store not configured")
				}
				if limit <= 0 {
					limit = 20
				}
				docs, err := s.ListPendingReviewSuggestions(ctx, limit)
				if err != nil {
					return err
				}
				if asJSON {
					return writeJSON(out, map[string]any{"entities": docs})
				}
				apiclient.WriteEntitySuggestionsTable(out, docs)
				return nil
			})
		},
	}
	c.Flags().Int64Var(&limit, "limit", 20, "max entities")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newEntityReclassifyCmd() *cobra.Command {
	var entityID string
	c := &cobra.Command{
		Use:   "reclassify",
		Short: "Queue Gemini entity re-classify (sets classify_force)",
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID = strings.TrimSpace(entityID)
			if entityID == "" {
				return fmt.Errorf("--id required")
			}
			return withLeadStore(cmd.Context(), cmd.OutOrStdout(), func(ctx context.Context, out io.Writer, s *store.LeadStore) error {
				if !s.EntitiesEnabled() {
					return fmt.Errorf("entity store not configured")
				}
				if err := s.MarkEntityClassifyForce(ctx, entityID); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "classify_force set for %s\n", entityID)
				return nil
			})
		},
	}
	c.Flags().StringVar(&entityID, "id", "", "entity_id")
	return c
}
