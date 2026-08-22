package warmpath

import (
	"context"
	"time"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/sink"
)

type mongoPendingScanner struct {
	store sink.StalePendingLister
}

// NewMongoPendingScanner adapts Mongo lead store queries to warm-path events.
func NewMongoPendingScanner(store sink.StalePendingLister) PendingLeadScanner {
	if store == nil {
		return nil
	}
	return &mongoPendingScanner{store: store}
}

func (m *mongoPendingScanner) ListStalePendingLeads(ctx context.Context, olderThan time.Duration, limit int) ([]Event, error) {
	docs, err := m.store.ListStalePendingLeads(ctx, olderThan, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(docs))
	for _, doc := range docs {
		out = append(out, EventFromLeadDoc(doc))
	}
	return out, nil
}

func (m *mongoPendingScanner) CountPendingAnalysis(ctx context.Context) (int64, error) {
	return m.store.CountPendingAnalysis(ctx)
}

type mongoDLQAdapter struct {
	writer sink.WarmAnalysisDLQWriter
}

// NewMongoAnalysisDLQ adapts Mongo DLQ writes to warm-path batch failures.
func NewMongoAnalysisDLQ(writer sink.WarmAnalysisDLQWriter) AnalysisDLQWriter {
	if writer == nil {
		return nil
	}
	return &mongoDLQAdapter{writer: writer}
}

func (d *mongoDLQAdapter) InsertWarmAnalysisFailures(ctx context.Context, batch []Event, attempts int, err error) error {
	if d == nil || d.writer == nil || len(batch) == 0 {
		return nil
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	failures := make([]sink.WarmAnalysisFailure, 0, len(batch))
	for _, ev := range batch {
		failures = append(failures, sink.WarmAnalysisFailure{
			HashID:   ev.HashID,
			Source:   ev.Source,
			Error:    msg,
			Attempts: attempts,
		})
	}
	return d.writer.InsertWarmAnalysisFailures(ctx, failures)
}

func storedContactsFormatted(contacts []sink.StoredContact) ([]string, []string) {
	// Match hot-path extract.FormatAll prefixes (telegram:, domain:, etc.) for Gemini batch input.
	extracted := make([]extract.Contact, 0, len(contacts))
	for _, c := range contacts {
		extracted = append(extracted, extract.Contact{
			Type:  c.Type,
			Value: c.Value,
		})
	}
	formatted := extract.FormatAll(extracted)
	types := make([]string, 0, len(contacts))
	for _, c := range contacts {
		if c.Type != "" {
			types = append(types, c.Type)
		}
	}
	return formatted, types
}
