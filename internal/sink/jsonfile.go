package sink

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bidshard/parser/internal/model"
)

// JSONFileSink appends lead JSON export records to a file.
type JSONFileSink struct {
	appendOnlyStore
	f      *os.File
	format string
}

func NewJSONFileSink(path, format string) (*JSONFileSink, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &JSONFileSink{f: f, format: ResolveExportFormat(path, format)}, nil
}

func (s *JSONFileSink) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	return s.appendLead(s.f, lead, s.format)
}

func (s *JSONFileSink) AppendLeadDoc(ctx context.Context, doc LeadDoc) error {
	_ = ctx
	return s.appendLead(s.f, LeadDocToModel(doc), s.format)
}
