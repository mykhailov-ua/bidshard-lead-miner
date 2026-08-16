package sink

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestBulkStoreFlushesTwoLeadsInOneTrip(t *testing.T) {
	t.Parallel()

	inner := NewMockBulkWriter()
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
	if inner.BulkCalls != 1 {
		t.Fatalf("bulk calls=%d want 1", inner.BulkCalls)
	}
	if inner.UpsertCalls != 2 {
		t.Fatalf("upsert calls=%d want 2", inner.UpsertCalls)
	}
}
