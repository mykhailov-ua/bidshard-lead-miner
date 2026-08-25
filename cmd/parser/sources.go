package main

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/sources"
	"github.com/spf13/cobra"
)

func newSourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sources",
		Short: "List available crawl sources",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Print source names and required env vars",
		Run: func(cmd *cobra.Command, args []string) {
			for _, info := range sources.Catalog() {
				inAll := ""
				if info.InAll {
					inAll = " [all]"
				}
				line := fmt.Sprintf("  %-10s%s", info.Name, inAll)
				if len(info.Requires) > 0 {
					line += fmt.Sprintf("  requires: %s", strings.Join(info.Requires, ", "))
				}
				if info.Note != "" {
					line += fmt.Sprintf("  (%s)", info.Note)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Set source: parser scan --source=forum   or   PARSER_SOURCE=forum")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Telegram is separate: parser telegram")
		},
	})

	cmd.AddCommand(newSourcesStatsCmd())

	return cmd
}

func newSourcesStatsCmd() *cobra.Command {
	var registryPath string
	c := &cobra.Command{
		Use:   "stats",
		Short: "Print unified source relevance scores from registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if registryPath == "" {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				registryPath = cfg.SourceRegistryPath
			}
			if registryPath == "" {
				registryPath = sourceregistry.DefaultPath
			}
			f, err := sourceregistry.Load(registryPath)
			if err != nil {
				return err
			}
			rows := sourceregistry.Summarize(f)
			sort.Slice(rows, func(i, j int) bool {
				if rows[i].RelevanceScore != rows[j].RelevanceScore {
					return rows[i].RelevanceScore > rows[j].RelevanceScore
				}
				return rows[i].Domain < rows[j].Domain
			})
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "DOMAIN\tSCORE\tTRIAGE\tTYPES\tDISCOVERED_BY")
			for _, row := range rows {
				_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
					row.Domain,
					row.RelevanceScore,
					row.TriageStatus,
					strings.Join(row.Types, ","),
					row.DiscoveredBy,
				)
			}
			return w.Flush()
		},
	}
	c.Flags().StringVar(&registryPath, "registry", "", "source registry path (default from config)")
	return c
}
