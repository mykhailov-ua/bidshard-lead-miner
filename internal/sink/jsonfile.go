package sink

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bidshard/parser/internal/model"
)

// JSONFileSink upserts lead JSON export records by hash_id.
type JSONFileSink struct {
	appendOnlyStore
	path   string
	format string
}

func NewJSONFileSink(path, format string) (*JSONFileSink, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &JSONFileSink{path: path, format: ResolveExportFormat(path, format)}, nil
}

func (s *JSONFileSink) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	return upsertLeadExport(s.path, lead, s.format)
}

func (s *JSONFileSink) AppendLeadDoc(ctx context.Context, doc LeadDoc) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	return upsertLeadExport(s.path, LeadDocToModel(doc), s.format)
}
