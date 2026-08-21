package sink

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type recordingBulkWriter struct {
	bulkCalls   int
	upsertCalls int
	records     map[string]struct{}
}

func newRecordingBulkWriter() *recordingBulkWriter {
	return &recordingBulkWriter{records: make(map[string]struct{})}
}

func (m *recordingBulkWriter) Exists(ctx context.Context, hashID string) (bool, error) {
	_, ok := m.records[hashID]
	return ok, nil
}

func (m *recordingBulkWriter) BulkUpsert(ctx context.Context, leads []model.Lead) error {
	m.bulkCalls++
	m.upsertCalls += len(leads)
	for _, lead := range leads {
		if lead.HashID != "" {
			m.records[lead.HashID] = struct{}{}
		}
	}
	return nil
}

func (m *recordingBulkWriter) Upsert(ctx context.Context, lead model.Lead) error {
	m.upsertCalls++
	if lead.HashID != "" {
		m.records[lead.HashID] = struct{}{}
	}
	return nil
}

func (m *recordingBulkWriter) UpdateStatus(ctx context.Context, hashID, status string) error {
	_ = ctx
	_ = hashID
	_ = status
	return nil
}

func TestBulkStoreFlushesTwoLeadsInOneTrip(t *testing.T) {
	t.Parallel()

	inner := newRecordingBulkWriter()
	store := NewBulkStore(inner, 50, time.Hour)

	leads := []model.Lead{
		{HashID: "a", Source: "ads_txt:one.com"},
		{HashID: "b", Source: "ads_txt:two.com"},
	}
	for _, lead := range leads {
		if err := store.Upsert(context.Background(), lead); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if inner.bulkCalls != 1 {
		t.Fatalf("bulk calls=%d want 1", inner.bulkCalls)
	}
	if inner.upsertCalls != 2 {
		t.Fatalf("upsert calls=%d want 2", inner.upsertCalls)
	}
}

type blockingBulkWriter struct {
	startOnce sync.Once
	started   chan struct{}
	release   chan struct{}
	records   map[string]struct{}
}

func newBlockingBulkWriter() *blockingBulkWriter {
	return &blockingBulkWriter{
		started: make(chan struct{}),
		release: make(chan struct{}),
		records: make(map[string]struct{}),
	}
}

func (m *blockingBulkWriter) Exists(ctx context.Context, hashID string) (bool, error) {
	_, ok := m.records[hashID]
	return ok, nil
}

func (m *blockingBulkWriter) BulkUpsert(ctx context.Context, leads []model.Lead) error {
	m.startOnce.Do(func() { close(m.started) })
	select {
	case <-m.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	for _, lead := range leads {
		if lead.HashID != "" {
			m.records[lead.HashID] = struct{}{}
		}
	}
	return nil
}

func (m *blockingBulkWriter) Upsert(ctx context.Context, lead model.Lead) error {
	_ = ctx
	if lead.HashID != "" {
		m.records[lead.HashID] = struct{}{}
	}
	return nil
}

func (m *blockingBulkWriter) UpdateStatus(ctx context.Context, hashID, status string) error {
	_ = ctx
	_ = hashID
	_ = status
	return nil
}

func TestBulkStoreUpsertDuringSlowFlush(t *testing.T) {
	t.Parallel()

	inner := newBlockingBulkWriter()
	store := NewBulkStore(inner, 50, time.Hour)
	if err := store.Upsert(context.Background(), model.Lead{HashID: "a", Source: "stub"}); err != nil {
		t.Fatal(err)
	}

	go func() {
		_ = store.Flush(context.Background())
	}()

	select {
	case <-inner.started:
	case <-time.After(time.Second):
		t.Fatal("bulk flush did not start")
	}

	done := make(chan error, 1)
	go func() {
		done <- store.Upsert(context.Background(), model.Lead{HashID: "b", Source: "stub"})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Upsert blocked during slow bulk flush")
	}

	close(inner.release)
}
