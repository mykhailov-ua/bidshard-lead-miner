package sink

import (
	"context"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type BulkWriter interface {
	BulkUpsert(ctx context.Context, leads []model.Lead) error
}

type BulkStore struct {
	inner         Store
	mu            sync.Mutex
	batch         []model.Lead
	batchSize     int
	flushInterval time.Duration
	lastFlush     time.Time
	FlushCount    int
}

func NewBulkStore(inner Store, batchSize int, flushInterval time.Duration) *BulkStore {
	if batchSize <= 0 {
		batchSize = 50
	}
	if flushInterval <= 0 {
		flushInterval = 2 * time.Second
	}
	return &BulkStore{
		inner:         inner,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		lastFlush:     time.Now(),
	}
}

func (b *BulkStore) Exists(ctx context.Context, hashID string) (bool, error) {
	b.mu.Lock()
	for _, lead := range b.batch {
		if lead.HashID == hashID {
			b.mu.Unlock()
			return true, nil
		}
	}
	b.mu.Unlock()
	return b.inner.Exists(ctx, hashID)
}

func (b *BulkStore) Upsert(ctx context.Context, lead model.Lead) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.batch = append(b.batch, lead)
	if len(b.batch) >= b.batchSize || time.Since(b.lastFlush) >= b.flushInterval {
		return b.flushLocked(ctx)
	}
	return nil
}

func (b *BulkStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	return b.inner.UpdateStatus(ctx, hashID, status)
}

func (b *BulkStore) Flush(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked(ctx)
}

func (b *BulkStore) flushLocked(ctx context.Context) error {
	if len(b.batch) == 0 {
		return nil
	}

	leads := append([]model.Lead(nil), b.batch...)
	b.batch = b.batch[:0]
	b.lastFlush = time.Now()
	b.FlushCount++

	if bw, ok := b.inner.(BulkWriter); ok {
		return bw.BulkUpsert(ctx, leads)
	}
	for _, lead := range leads {
		if err := b.inner.Upsert(ctx, lead); err != nil {
			return err
		}
	}
	return nil
}

type MockBulkWriter struct {
	BulkCalls   int
	UpsertCalls int
	records     map[string]struct{}
}

func NewMockBulkWriter() *MockBulkWriter {
	return &MockBulkWriter{records: make(map[string]struct{})}
}

func (m *MockBulkWriter) Exists(ctx context.Context, hashID string) (bool, error) {
	_, ok := m.records[hashID]
	return ok, nil
}

func (m *MockBulkWriter) BulkUpsert(ctx context.Context, leads []model.Lead) error {
	m.BulkCalls++
	m.UpsertCalls += len(leads)
	for _, lead := range leads {
		if lead.HashID != "" {
			m.records[lead.HashID] = struct{}{}
		}
	}
	return nil
}

func (m *MockBulkWriter) Upsert(ctx context.Context, lead model.Lead) error {
	m.UpsertCalls++
	if lead.HashID != "" {
		m.records[lead.HashID] = struct{}{}
	}
	return nil
}

func (m *MockBulkWriter) UpdateStatus(ctx context.Context, hashID, status string) error {
	_ = ctx
	_ = hashID
	_ = status
	return nil
}
