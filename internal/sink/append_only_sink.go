package sink

import (
	"context"
	"io"
	"sync"

	"github.com/bidshard/parser/internal/model"
)

type appendOnlyStore struct {
	mu sync.Mutex
}

func (s *appendOnlyStore) Exists(ctx context.Context, hashID string) (bool, error) {
	_ = ctx
	_ = hashID
	return false, nil
}

func (s *appendOnlyStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	_ = ctx
	_ = hashID
	_ = status
	return nil
}

func (s *appendOnlyStore) appendLead(w io.Writer, lead model.Lead, format string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return EncodeLeadExport(w, lead, format)
}
