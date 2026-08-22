package warmpath

import (
	"context"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// PendingLeadScanner loads stale deferred leads for warm-path rescan.
type PendingLeadScanner interface {
	ListStalePendingLeads(ctx context.Context, olderThan time.Duration, limit int) ([]Event, error)
	CountPendingAnalysis(ctx context.Context) (int64, error)
}

// AnalysisDLQWriter persists warm-path batch failures after retries are exhausted.
type AnalysisDLQWriter interface {
	InsertWarmAnalysisFailures(ctx context.Context, batch []Event, attempts int, err error) error
}

// EventFromLeadDoc rebuilds a warm-path queue event from a stored lead row.
// Domain is omitted: LeadDoc has no domain field; RDAP country and contacts still feed Gemini.
func EventFromLeadDoc(doc sink.LeadDoc) Event {
	contacts, types := storedContactsFormatted(doc.Contacts)
	return Event{
		HashID:        doc.HashID,
		RoundID:       doc.RoundID,
		Source:        doc.Source,
		Title:         doc.Title,
		Snippet:       doc.Snippet,
		Contacts:      contacts,
		ContactTypes:  types,
		Stack:         append([]string(nil), doc.Stack...),
		Score:         doc.Score,
		Priority:      doc.Priority,
		Matched:       append([]string(nil), doc.Matched...),
		RDAPCountry:   doc.WhoisCountry,
		DomainAgeDays: doc.DomainAgeDays,
		DisplayName:   doc.DisplayName,
		EntityID:      doc.EntityID,
		EntityHeat:    doc.EntityHeat,
		HeatTier:      doc.HeatTier,
		InlineICP:     doc.ICP,
		TS:            doc.TS,
	}
}
