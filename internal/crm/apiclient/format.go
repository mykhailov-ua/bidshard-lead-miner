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
	if len(stats.ByOutcome) > 0 {
		_, _ = fmt.Fprintln(out, "outcomes:")
		for _, row := range stats.ByOutcome {
			outcome := row.Status
			if outcome == "" {
				outcome = "(empty)"
			}
			_, _ = fmt.Fprintf(out, "  %s: %d\n", outcome, row.Count)
		}
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
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\n",
			entity.DisplayHeatScore(doc),
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

func WriteEntityInboxTable(out io.Writer, cards []entity.InboxCard) {
	if len(cards) == 0 {
		_, _ = fmt.Fprintln(out, "no entities in inbox")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "HEAT\tTIER\tSIGHT\tSRC\tNEW\tPEND\tENTITY_ID\tNARRATIVE")
	for _, card := range cards {
		narrative := card.OutreachAngle
		if narrative == "" {
			narrative = card.UnifiedPain
		}
		if narrative == "" {
			narrative = card.EntityProof
		}
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%d\t%d\t%s\t%s\n",
			card.HeatScore,
			card.HeatTier,
			card.SightingCount,
			card.SourceCount,
			card.NewLeadCount,
			card.PendingSuggestions,
			card.EntityID,
			truncate(narrative, 56),
		)
	}
	_ = tw.Flush()
}

func WriteEntityInboxCard(out io.Writer, card entity.InboxCard) {
	_, _ = fmt.Fprintf(out, "entity_id: %s\n", card.EntityID)
	_, _ = fmt.Fprintf(out, "heat: %d (%s) sightings=%d sources=%d engage_priority=%d\n",
		card.HeatScore, card.HeatTier, card.SightingCount, card.SourceCount, card.EngagePriority)
	if len(card.SourceFamilies) > 0 {
		_, _ = fmt.Fprintf(out, "source_families: %s\n", strings.Join(card.SourceFamilies, ", "))
	}
	if card.NewLeadCount > 0 {
		_, _ = fmt.Fprintf(out, "new_leads: %d\n", card.NewLeadCount)
	}
	if card.PendingSuggestions > 0 {
		_, _ = fmt.Fprintf(out, "pending_suggestions: %d\n", card.PendingSuggestions)
	}
	if card.NeedsReview {
		_, _ = fmt.Fprintln(out, "needs_review: true")
	}
	if card.ClassifyForce {
		_, _ = fmt.Fprintln(out, "classify_force: true")
	}
	if pain := strings.TrimSpace(card.UnifiedPain); pain != "" {
		_, _ = fmt.Fprintf(out, "unified_pain: %s\n", pain)
	}
	if intent := strings.TrimSpace(card.BuyerIntent); intent != "" {
		_, _ = fmt.Fprintf(out, "buyer_intent: %s\n", intent)
	}
	if proof := strings.TrimSpace(card.EntityProof); proof != "" {
		_, _ = fmt.Fprintf(out, "entity_proof: %s\n", proof)
	}
	if angle := strings.TrimSpace(card.OutreachAngle); angle != "" {
		_, _ = fmt.Fprintf(out, "outreach_angle: %s\n", angle)
	}
	if ch := strings.TrimSpace(card.BestContactChannel); ch != "" {
		_, _ = fmt.Fprintf(out, "best_contact_channel: %s\n", ch)
	}
}

func WriteEntitySuggestionsTable(out io.Writer, docs []entity.EntityDoc) {
	if len(docs) == 0 {
		_, _ = fmt.Fprintln(out, "no pending suggestions")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ENTITY_ID\tPEER\tACTION\tSHARED\tWHY")
	for _, doc := range docs {
		for _, s := range doc.ReviewSuggestions {
			if !strings.EqualFold(strings.TrimSpace(s.Status), "pending") {
				continue
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				doc.EntityID,
				s.PeerEntityID,
				s.Action,
				truncate(s.SharedDomain, 32),
				truncate(s.Why, 48),
			)
		}
	}
	_ = tw.Flush()
}

func WriteLeadTable(out io.Writer, leads []sink.LeadDoc) {
	if len(leads) == 0 {
		_, _ = fmt.Fprintln(out, "no leads")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "SCORE\tOUTCOME\tACTION\tCHANNEL\tSTATUS\tSOURCE\tGEO\tSNIPPET\tHASH")
	for _, lead := range leads {
		geo := lead.GeoCountry
		if geo == "" {
			geo = lead.CompanyCountry
		}
		if geo == "" {
			geo = "-"
		}
		outcome := strings.TrimSpace(lead.Outcome)
		if outcome == "" {
			outcome = "-"
		}
		action := strings.TrimSpace(lead.NextAction)
		if action == "" {
			action = "-"
		}
		channel := strings.TrimSpace(lead.ContactChannel)
		if channel == "" {
			channel = "-"
		}
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			lead.Score,
			outcome,
			action,
			channel,
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
