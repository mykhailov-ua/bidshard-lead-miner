package sink

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/bidshard/parser/internal/model"
)

// JSONFileSink appends one JSON object per line for each new accepted lead.
type JSONFileSink struct {
	mu  sync.Mutex
	f   *os.File
	enc *json.Encoder
}

func NewJSONFileSink(path string) (*JSONFileSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &JSONFileSink{f: f, enc: json.NewEncoder(f)}, nil
}

func (s *JSONFileSink) Exists(ctx context.Context, hashID string) (bool, error) {
	_ = ctx
	_ = hashID
	return false, nil
}

func (s *JSONFileSink) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(LeadJSONMap(lead))
}

func (s *JSONFileSink) UpdateStatus(ctx context.Context, hashID, status string) error {
	_ = ctx
	_ = hashID
	_ = status
	return nil
}
