package sink

import (
	"context"

	"github.com/bidshard/parser/internal/model"
)

type Store interface {
	Exists(ctx context.Context, hashID string) (bool, error)
	Upsert(ctx context.Context, lead model.Lead) error
}

type StubStore struct {
	ExistsCalls int
	records     map[string]struct{}
}

func NewStubStore() *StubStore {
	return &StubStore{records: make(map[string]struct{})}
}

func (s *StubStore) Exists(ctx context.Context, hashID string) (bool, error) {
	_ = ctx
	s.ExistsCalls++
	_, ok := s.records[hashID]
	return ok, nil
}

func (s *StubStore) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	if lead.HashID != "" {
		s.records[lead.HashID] = struct{}{}
	}
	return nil
}

func (s *StubStore) Seed(hashID string) {
	s.records[hashID] = struct{}{}
}
