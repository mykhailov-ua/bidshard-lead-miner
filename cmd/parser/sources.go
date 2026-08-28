package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/pretty"
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
			printSourcesList(cmd.OutOrStdout(), sources.Catalog())
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
			out := cmd.OutOrStdout()
			color := cliColor(out)
			pretty.Section(out, color, "Source registry relevance")
			header := []string{"domain", "score", "triage", "types", "discovered_by"}
			table := make([][]string, 0, len(rows))
			for _, row := range rows {
				table = append(table, []string{
					row.Domain,
					fmt.Sprintf("%d", row.RelevanceScore),
					row.TriageStatus,
					strings.Join(row.Types, ","),
					row.DiscoveredBy,
				})
			}
			pretty.PrintTable(out, color, header, table)
			return nil
		},
	}
	c.Flags().StringVar(&registryPath, "registry", "", "source registry path (default from config)")
	return c
}
