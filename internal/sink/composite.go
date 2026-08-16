package sink

import (
	"context"

	"github.com/bidshard/parser/internal/model"
)

// CompositeStore fans out writes; Exists is true if any backing store has the hash_id.
type CompositeStore struct {
	stores []Store
}

func NewCompositeStore(stores ...Store) *CompositeStore {
	return &CompositeStore{stores: stores}
}

func (c *CompositeStore) Exists(ctx context.Context, hashID string) (bool, error) {
	for _, s := range c.stores {
		exists, err := s.Exists(ctx, hashID)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}

func (c *CompositeStore) Upsert(ctx context.Context, lead model.Lead) error {
	for _, s := range c.stores {
		if err := s.Upsert(ctx, lead); err != nil {
			return err
		}
	}
	return nil
}
