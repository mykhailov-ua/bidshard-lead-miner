package apiclient

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/sink"
)

func WriteStats(out io.Writer, stats store.DBStats) {
	_, _ = fmt.Fprintf(out, "leads total: %d\n", stats.TotalLeads)
	for _, row := range stats.ByStatus {
		status := row.Status
		if status == "" {
			status = "(empty)"
		}
		_, _ = fmt.Fprintf(out, "  %s: %d\n", status, row.Count)
	}
}

func WriteEntityTable(out io.Writer, entities []entity.EntityDoc) {
	if len(entities) == 0 {
		_, _ = fmt.Fprintln(out, "no entities")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "HEAT\tTIER\tFAMILIES\tPAIN\tENTITY_ID")
	for _, doc := range entities {
		pain := doc.UnifiedPain
		if pain == "" && len(doc.Matched) > 0 {
			pain = strings.Join(doc.Matched, ",")
		}
		_, _ = fmt.Fprintf(tw, "%.0f\t%s\t%d\t%s\t%s\n",
			doc.HeatScore,
			doc.HeatTier,
			doc.SourceCount,
			truncate(pain, 48),
			doc.EntityID,
		)
	}
	_ = tw.Flush()
}

func WriteEntityJSON(out io.Writer, doc entity.EntityDoc) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

func WriteLeadTable(out io.Writer, leads []sink.LeadDoc) {
	if len(leads) == 0 {
		_, _ = fmt.Fprintln(out, "no leads")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "SCORE\tSTATUS\tSOURCE\tGEO\tSNIPPET\tHASH")
	for _, lead := range leads {
		geo := lead.GeoCountry
		if geo == "" {
			geo = lead.CompanyCountry
		}
		if geo == "" {
			geo = "-"
		}
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			lead.Score,
			strings.TrimSpace(lead.Status),
			lead.Source,
			geo,
			truncate(lead.Snippet, 72),
			lead.HashID,
		)
	}
	_ = tw.Flush()
}

func WriteLeadJSON(out io.Writer, lead sink.LeadDoc) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(lead)
}

func truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
