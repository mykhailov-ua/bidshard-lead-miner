package sink

import (
	"context"
	"io"

	"github.com/bidshard/parser/internal/model"
)

type NDJSONSink struct {
	appendOnlyStore
	w io.Writer
}

func NewNDJSONSink(w io.Writer) *NDJSONSink {
	return &NDJSONSink{w: w}
}

func (s *NDJSONSink) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	return s.appendLead(s.w, lead, ExportFormatNDJSON)
}
