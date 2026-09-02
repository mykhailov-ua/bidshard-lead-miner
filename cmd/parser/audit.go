package main

import (
	"fmt"
	"os"

	"github.com/bidshard/parser/internal/audit"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Offline export audits (no network)",
	}
	cmd.AddCommand(newAuditProgrammaticCmd())
	return cmd
}

func newAuditProgrammaticCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "programmatic <leads.jsonl>",
		Short: "Report rows that would drop under programmatic vertical gate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			rep, err := audit.ScanLeadsJSONL(path)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(os.Stdout, audit.FormatProgrammaticReport(rep))
			return err
		},
	}
}
