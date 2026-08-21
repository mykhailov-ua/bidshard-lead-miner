package sink

import (
	"context"
	"sync"
	"sync/atomic"
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
	leadsWritten  atomic.Int64
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

func (b *BulkStore) LeadsWritten() int64 {
	return b.leadsWritten.Load()
}

// UnderlyingStore returns the store wrapped by BulkStore (Mongo, export, webhook, etc.).
func (b *BulkStore) UnderlyingStore() Store {
	if b == nil {
		return nil
	}
	return b.inner
}

func (b *BulkStore) Exists(ctx context.Context, hashID string) (bool, error) {
	b.mu.Lock()
	// Treat pending batch as written to avoid duplicate upserts within the same crawl flush window.
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
	var flushBatch []model.Lead
	b.mu.Lock()
	b.batch = append(b.batch, lead)
	if len(b.batch) >= b.batchSize || time.Since(b.lastFlush) >= b.flushInterval {
		// Copy batch and release mu before Mongo I/O so Exists/Upsert from workers are not blocked.
		flushBatch = append([]model.Lead(nil), b.batch...)
		b.batch = b.batch[:0]
		b.lastFlush = time.Now()
		b.FlushCount++
	}
	b.mu.Unlock()
	if flushBatch != nil {
		return b.flushBatch(ctx, flushBatch)
	}
	return nil
}

func (b *BulkStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	return b.inner.UpdateStatus(ctx, hashID, status)
}

func (b *BulkStore) ApplyEntityHeat(ctx context.Context, hashID string, patch EntityHeatPatch) error {
	if patcher, ok := b.inner.(EntityHeatPatcher); ok {
		return patcher.ApplyEntityHeat(ctx, hashID, patch)
	}
	return nil
}

func (b *BulkStore) ApplyCrossSourceHot(ctx context.Context, hashID string, boost int) error {
	if patcher, ok := b.inner.(CrossSourceHotPatcher); ok {
		return patcher.ApplyCrossSourceHot(ctx, hashID, boost)
	}
	return nil
}

func (b *BulkStore) Flush(ctx context.Context) error {
	b.mu.Lock()
	var flushBatch []model.Lead
	if len(b.batch) > 0 {
		flushBatch = append([]model.Lead(nil), b.batch...)
		b.batch = b.batch[:0]
		b.lastFlush = time.Now()
		b.FlushCount++
	}
	b.mu.Unlock()
	if flushBatch == nil {
		return nil
	}
	return b.flushBatch(ctx, flushBatch)
}

func (b *BulkStore) flushBatch(ctx context.Context, leads []model.Lead) error {
	if len(leads) == 0 {
		return nil
	}

	if bw, ok := b.inner.(BulkWriter); ok {
		if err := bw.BulkUpsert(ctx, leads); err != nil {
			return err
		}
		b.leadsWritten.Add(int64(len(leads)))
		return nil
	}
	// Fallback for stores without BulkWriter: upsert one lead at a time.
	for _, lead := range leads {
		if err := b.inner.Upsert(ctx, lead); err != nil {
			return err
		}
		b.leadsWritten.Add(1)
	}
	return nil
}
