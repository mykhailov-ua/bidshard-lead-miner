package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/bidshard/parser/internal/crm/apiclient"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/sink"
	"github.com/spf13/cobra"
)

func newAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Remote CRM admin over HTTP (VPS behind Caddy)",
		Long: `Call /v1/admin/* on a remote crm-bot instance.

Env:
  CRM_API_URL=https://crm.example.com
  CRM_API_USER=sales
  CRM_API_PASSWORD=...

Local dev without Caddy auth:
  CRM_API_URL=http://127.0.0.1:8080

Examples:
  crm-bot api stats
  crm-bot api list --status new --limit 20
  crm-bot api search --q voluum
  crm-bot api show --hash abcdef12
  crm-bot api set-status --hash abcdef12 --status contacted
  crm-bot api delete --hash abcdef12 --yes
  crm-bot api purge --status spam --yes`,
	}
	cmd.AddCommand(
		newAPIStatsCmd(),
		newAPIListCmd(),
		newAPISearchCmd(),
		newAPIShowCmd(),
		newAPIEntitiesCmd(),
		newAPISetStatusCmd(),
		newAPIDeleteCmd(),
		newAPIPurgeCmd(),
	)
	return cmd
}

func newAPIStatsCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "stats",
		Short: "Print lead counts by status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				var stats store.DBStats
				if err := client.GetJSON(ctx, "/v1/admin/stats", &stats); err != nil {
					return err
				}
				if asJSON {
					return writeJSON(cmd.OutOrStdout(), stats)
				}
				apiclient.WriteStats(cmd.OutOrStdout(), stats)
				return nil
			})
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newAPIListCmd() *cobra.Command {
	var (
		status       string
		sourcePrefix string
		scoreMax     int
		limit        int
		sortBy       string
		asJSON       bool
		skipInbox    bool
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List leads",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				qs := url.Values{}
				if limit <= 0 {
					limit = 50
				}
				qs.Set("limit", strconv.Itoa(limit))
				if s := strings.TrimSpace(status); s != "" {
					qs.Set("status", s)
				}
				if p := strings.TrimSpace(sourcePrefix); p != "" {
					qs.Set("source_prefix", p)
				}
				if scoreMax > 0 {
					qs.Set("score_max", strconv.Itoa(scoreMax))
				}
				if skipInbox {
					qs.Set("inbox", "false")
				} else if strings.EqualFold(strings.TrimSpace(status), "new") {
					qs.Set("inbox", "true")
				}
				if s := strings.TrimSpace(sortBy); s != "" {
					qs.Set("sort", s)
				}
				var result store.ListResult
				path := "/v1/admin/leads?" + qs.Encode()
				if err := client.GetJSON(ctx, path, &result); err != nil {
					return err
				}
				if asJSON {
					return writeJSON(cmd.OutOrStdout(), result)
				}
				apiclient.WriteLeadTable(cmd.OutOrStdout(), result.Leads)
				return nil
			})
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status")
	c.Flags().StringVar(&sourcePrefix, "source-prefix", "", "filter by source prefix")
	c.Flags().IntVar(&scoreMax, "score-max", 0, "filter score <= N")
	c.Flags().IntVar(&limit, "limit", 50, "max rows")
	c.Flags().StringVar(&sortBy, "sort", "", "sort: heat (default for status=new) or score")
	c.Flags().BoolVar(&skipInbox, "all", false, "include pending/deferred leads (skip inbox filter)")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newAPISearchCmd() *cobra.Command {
	var (
		query  string
		limit  int
		asJSON bool
	)
	c := &cobra.Command{
		Use:   "search",
		Short: "Search leads by snippet or source",
		RunE: func(cmd *cobra.Command, args []string) error {
			query = strings.TrimSpace(query)
			if query == "" {
				return fmt.Errorf("--q required")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				if limit <= 0 {
					limit = 20
				}
				qs := url.Values{"q": {query}, "limit": {strconv.Itoa(limit)}}
				var result store.SearchResult
				if err := client.GetJSON(ctx, "/v1/admin/leads/search?"+qs.Encode(), &result); err != nil {
					return err
				}
				if asJSON {
					return writeJSON(cmd.OutOrStdout(), result)
				}
				apiclient.WriteLeadTable(cmd.OutOrStdout(), result.Leads)
				if result.Truncated {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(truncated)")
				}
				return nil
			})
		},
	}
	c.Flags().StringVar(&query, "q", "", "search text")
	c.Flags().IntVar(&limit, "limit", 20, "max rows")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newAPIEntitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entities",
		Short: "Entity graph admin (heat-ranked buyers)",
	}
	cmd.AddCommand(newAPIEntitiesListCmd(), newAPIEntitiesGetCmd(), newAPIEntitiesLeadsCmd())
	return cmd
}

func newAPIEntitiesListCmd() *cobra.Command {
	var minTier string
	var limit int
	var asJSON bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List entities by heat tier",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				if limit <= 0 {
					limit = 20
				}
				qs := url.Values{"limit": {strconv.Itoa(limit)}}
				if t := strings.TrimSpace(minTier); t != "" {
					qs.Set("min_tier", t)
				}
				var result store.EntityListResult
				if err := client.GetJSON(ctx, "/v1/admin/entities/list?"+qs.Encode(), &result); err != nil {
					return err
				}
				if asJSON {
					return writeJSON(cmd.OutOrStdout(), result)
				}
				apiclient.WriteEntityTable(cmd.OutOrStdout(), result.Entities)
				return nil
			})
		},
	}
	c.Flags().StringVar(&minTier, "min-tier", "hot", "minimum heat tier: warm|hot|blazing")
	c.Flags().IntVar(&limit, "limit", 20, "max rows")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newAPIEntitiesGetCmd() *cobra.Command {
	var entityID string
	var asJSON bool
	c := &cobra.Command{
		Use:   "get",
		Short: "Show one entity graph node",
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID = strings.TrimSpace(entityID)
			if entityID == "" {
				return fmt.Errorf("--id required")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				qs := url.Values{"entity_id": {entityID}}
				var doc entity.EntityDoc
				if err := client.GetJSON(ctx, "/v1/admin/entities/get?"+qs.Encode(), &doc); err != nil {
					return err
				}
				if asJSON {
					return writeJSON(cmd.OutOrStdout(), doc)
				}
				return apiclient.WriteEntityJSON(cmd.OutOrStdout(), doc)
			})
		},
	}
	c.Flags().StringVar(&entityID, "id", "", "entity_id")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newAPIEntitiesLeadsCmd() *cobra.Command {
	var entityID string
	var limit int
	var asJSON bool
	c := &cobra.Command{
		Use:   "leads",
		Short: "List leads linked to an entity",
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID = strings.TrimSpace(entityID)
			if entityID == "" {
				return fmt.Errorf("--id required")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				if limit <= 0 {
					limit = 20
				}
				qs := url.Values{"entity_id": {entityID}, "limit": {strconv.Itoa(limit)}}
				var result struct {
					EntityID string         `json:"entity_id"`
					Leads    []sink.LeadDoc `json:"leads"`
				}
				if err := client.GetJSON(ctx, "/v1/admin/entities/leads?"+qs.Encode(), &result); err != nil {
					return err
				}
				if asJSON {
					return writeJSON(cmd.OutOrStdout(), result)
				}
				apiclient.WriteLeadTable(cmd.OutOrStdout(), result.Leads)
				return nil
			})
		},
	}
	c.Flags().StringVar(&entityID, "id", "", "entity_id")
	c.Flags().IntVar(&limit, "limit", 20, "max rows")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return c
}

func newAPIShowCmd() *cobra.Command {
	var hashID string
	c := &cobra.Command{
		Use:   "show",
		Short: "Show one lead as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			hashID = strings.TrimSpace(hashID)
			if hashID == "" {
				return fmt.Errorf("--hash required")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				qs := url.Values{"hash_id": {hashID}}
				var lead sink.LeadDoc
				if err := client.GetJSON(ctx, "/v1/admin/leads/get?"+qs.Encode(), &lead); err != nil {
					return err
				}
				return apiclient.WriteLeadJSON(cmd.OutOrStdout(), lead)
			})
		},
	}
	c.Flags().StringVar(&hashID, "hash", "", "lead hash_id or unique prefix")
	return c
}

func newAPISetStatusCmd() *cobra.Command {
	var hashID, status string
	c := &cobra.Command{
		Use:   "set-status",
		Short: "Update lead status",
		RunE: func(cmd *cobra.Command, args []string) error {
			hashID = strings.TrimSpace(hashID)
			if hashID == "" {
				return fmt.Errorf("--hash required")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				var out map[string]string
				if err := client.PatchJSON(ctx, "/v1/admin/leads", map[string]string{
					"hash_id": hashID,
					"status":  status,
				}, &out); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "updated hash_id=%s status=%s\n", out["hash_id"], out["status"])
				return nil
			})
		},
	}
	c.Flags().StringVar(&hashID, "hash", "", "lead hash_id or unique prefix")
	c.Flags().StringVar(&status, "status", "", "new|contacted|won|lost|spam|archived")
	return c
}

func newAPIDeleteCmd() *cobra.Command {
	var hashID string
	var yes bool
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete one lead",
		RunE: func(cmd *cobra.Command, args []string) error {
			hashID = strings.TrimSpace(hashID)
			if hashID == "" {
				return fmt.Errorf("--hash required")
			}
			if !yes {
				return fmt.Errorf("pass --yes to confirm delete")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				qs := url.Values{"hash_id": {hashID}}
				var result store.DeleteResult
				if err := client.DeleteJSON(ctx, "/v1/admin/leads?"+qs.Encode(), &result); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted leads=%d notes=%d meta=%d\n", result.Leads, result.Notes, result.Meta)
				return nil
			})
		},
	}
	c.Flags().StringVar(&hashID, "hash", "", "lead hash_id or unique prefix")
	c.Flags().BoolVar(&yes, "yes", false, "confirm delete")
	return c
}

func newAPIPurgeCmd() *cobra.Command {
	var (
		all          bool
		status       string
		sourcePrefix string
		scoreMax     int
		yes          bool
	)
	c := &cobra.Command{
		Use:   "purge",
		Short: "Bulk delete leads matching filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("pass --yes to confirm purge")
			}
			return withAPIClient(cmd.Context(), func(ctx context.Context, client *apiclient.Client) error {
				body := map[string]any{
					"confirm":       "purge",
					"all":           all,
					"status":        strings.TrimSpace(status),
					"source_prefix": strings.TrimSpace(sourcePrefix),
					"score_max":     scoreMax,
				}
				var result store.DeleteResult
				if err := client.PostJSON(ctx, "/v1/admin/leads/purge", body, &result); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "purged leads=%d notes=%d meta=%d\n", result.Leads, result.Notes, result.Meta)
				return nil
			})
		},
	}
	c.Flags().BoolVar(&all, "all", false, "delete all leads")
	c.Flags().StringVar(&status, "status", "", "lead status filter")
	c.Flags().StringVar(&sourcePrefix, "source-prefix", "", "source prefix filter")
	c.Flags().IntVar(&scoreMax, "score-max", 0, "delete leads with score <= N")
	c.Flags().BoolVar(&yes, "yes", false, "confirm purge")
	return c
}

func withAPIClient(ctx context.Context, fn func(context.Context, *apiclient.Client) error) error {
	cfg, err := apiclient.LoadConfig()
	if err != nil {
		return err
	}
	return fn(ctx, apiclient.New(cfg))
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
