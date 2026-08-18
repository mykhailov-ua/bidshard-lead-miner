package sink

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/bidshard/parser/internal/model"
)

type NDJSONSink struct {
	mu sync.Mutex
	w  io.Writer
}

func NewNDJSONSink(w io.Writer) *NDJSONSink {
	return &NDJSONSink{w: w}
}

func (s *NDJSONSink) Exists(ctx context.Context, hashID string) (bool, error) {
	_ = ctx
	_ = hashID
	return false, nil
}

func (s *NDJSONSink) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.NewEncoder(s.w).Encode(LeadJSONMap(lead))
}

func (s *NDJSONSink) UpdateStatus(ctx context.Context, hashID, status string) error {
	_ = ctx
	_ = hashID
	_ = status
	return nil
}
