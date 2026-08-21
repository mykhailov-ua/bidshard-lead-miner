package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/pretty"
	"github.com/bidshard/parser/internal/sink"
)

func resolveOutputMode(mode string, w io.Writer) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "auto":
		if pretty.IsTerminal(w) {
			return "pretty"
		}
		return "quiet"
	case "table":
		return "pretty"
	case "json", "ndjson":
		return "ndjson"
	case "json-pretty", "pretty", "quiet":
		return strings.ToLower(mode)
	default:
		return strings.ToLower(mode)
	}
}

func (r *Reporter) writePretty(stats pipeline.RoundStats) {
	w := r.stdout
	color := pretty.ColorEnabled(w)
	totalSources := stats.SourcesOK + stats.SourcesFail

	pretty.Divider(w, color)
	pretty.Header(w, color, fmt.Sprintf("scan round %s (%s)", stats.RoundID, stats.Duration.Round(time.Millisecond*100)))
	pretty.KV(w, color, "sources", fmt.Sprintf("%d/%d ok", stats.SourcesOK, totalSources))
	pretty.KV(w, color, "raw", fmt.Sprintf("%d", stats.RawTotal))
	pretty.KV(w, color, "accepted", fmt.Sprintf("%d", stats.Accepted))
	if stats.RejectedGeo > 0 {
		pretty.KV(w, color, "rejected_geo", fmt.Sprintf("%d", stats.RejectedGeo))
	}
	if stats.Dedup > 0 {
		pretty.KV(w, color, "dedup", fmt.Sprintf("%d", stats.Dedup))
	}
	if stats.Dropped > 0 {
		pretty.KV(w, color, "dropped", fmt.Sprintf("%d", stats.Dropped))
	}
	pretty.KV(w, color, "priority", fmt.Sprintf("high %d  medium %d  low %d", stats.High, stats.Medium, stats.Low))

	leads := stats.Leads
	if len(leads) == 0 {
		pretty.KV(w, color, "leads", "none this round")
		pretty.Divider(w, color)
		return
	}

	for _, lead := range leads {
		r.writePrettyLead(w, color, lead)
	}
	pretty.Divider(w, color)
}

func (r *Reporter) writePrettyLead(w io.Writer, color bool, lead model.Lead) {
	doc := sink.LeadExport(lead)
	label, style := pretty.PriorityStyle(doc.Priority)
	title := fmt.Sprintf("%s  score %d  %s", label, doc.Score, doc.Source)
	if color {
		_, _ = fmt.Fprintf(w, "\n%s* %s%s\n", style, title, pretty.Reset)
	} else {
		_, _ = fmt.Fprintf(w, "\n* %s\n", title)
	}

	if matched := strings.Join(doc.Matched, ", "); matched != "" {
		pretty.KV(w, color, "matched", matched)
	}
	if contacts := formatContacts(doc.Contacts); contacts != "" {
		pretty.KV(w, color, "contacts", contacts)
	}
	if doc.Snippet != "" {
		pretty.KV(w, color, "snippet", truncateSnippet(doc.Snippet, 160))
	}
	if doc.HashID != "" {
		pretty.KV(w, color, "hash_id", doc.HashID)
	}
	if doc.ICP != "" {
		pretty.KV(w, color, "icp", doc.ICP)
	}
	if doc.GeoCountry != "" {
		pretty.KV(w, color, "geo", doc.GeoCountry)
	}
	if doc.OutreachChannel != "" {
		pretty.KV(w, color, "outreach", fmt.Sprintf("%s - %s", doc.OutreachChannel, doc.OutreachAngle))
	}
	if doc.OutreachDraft != "" {
		pretty.KV(w, color, "draft", truncateSnippet(doc.OutreachDraft, 120))
	}
}

func formatContacts(contacts []sink.StoredContact) string {
	if len(contacts) == 0 {
		return ""
	}
	parts := make([]string, 0, len(contacts))
	for _, c := range contacts {
		parts = append(parts, c.Type+":"+c.Value)
	}
	return strings.Join(parts, ", ")
}

func truncateSnippet(snippet string, max int) string {
	snippet = strings.Join(strings.Fields(snippet), " ")
	if len(snippet) <= max {
		return snippet
	}
	return snippet[:max-3] + "..."
}

func (r *Reporter) writeJSONPretty(stats pipeline.RoundStats) {
	for _, lead := range stats.Leads {
		_ = sink.EncodeLeadExport(r.stdout, lead, sink.ExportFormatPretty)
	}
}
