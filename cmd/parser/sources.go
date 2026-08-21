package main

import (
	"fmt"
	"strings"

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

	return cmd
}
