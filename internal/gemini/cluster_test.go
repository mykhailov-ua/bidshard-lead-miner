package gemini

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

type memEmbedStore struct {
	mu   sync.Mutex
	docs []sink.EmbeddingDoc
}

func (m *memEmbedStore) Upsert(_ context.Context, doc sink.EmbeddingDoc) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.docs {
		if existing.Key == doc.Key {
			m.docs[i] = doc
			return nil
		}
	}
	m.docs = append(m.docs, doc)
	return nil
}

func (m *memEmbedStore) RecentVectorsByKind(_ context.Context, kind string, limit int) ([]sink.EmbeddingDoc, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []sink.EmbeddingDoc
	for i := len(m.docs) - 1; i >= 0 && len(out) < limit; i-- {
		doc := m.docs[i]
		if kind != "" && doc.Kind != kind {
			continue
		}
		out = append(out, doc)
	}
	return out, nil
}

func TestLeadClusterDuplicate(t *testing.T) {
	t.Parallel()

	vec := []float32{1, 0, 0}
	embedder := stubEmbedder{vectors: map[string][]float32{
		"voluum alternative postback failing": vec,
	}}
	store := &memEmbedStore{}
	cluster := NewLeadCluster(embedder, store, 0.9)

	if err := cluster.Record(context.Background(), "hash-a", "voluum alternative postback failing"); err != nil {
		t.Fatal(err)
	}

	dup, clusterOf, err := cluster.CheckDuplicate(context.Background(), "hash-b", "voluum alternative postback failing")
	if err != nil {
		t.Fatal(err)
	}
	if !dup || clusterOf != "hash-a" {
		t.Fatalf("dup=%v clusterOf=%q", dup, clusterOf)
	}

	dup, _, err = cluster.CheckDuplicate(context.Background(), "hash-a", "voluum alternative postback failing")
	if err != nil {
		t.Fatal(err)
	}
	if dup {
		t.Fatal("expected same hash_id not to be duplicate")
	}
}

func TestLeadClusterRecordUpsert(t *testing.T) {
	t.Parallel()

	store := &memEmbedStore{}
	cluster := NewLeadCluster(stubEmbedder{vectors: map[string][]float32{
		"text": {0, 1, 0},
	}}, store, 0.9)

	if err := cluster.Record(context.Background(), "abc", "text"); err != nil {
		t.Fatal(err)
	}
	if len(store.docs) != 1 {
		t.Fatalf("docs=%d", len(store.docs))
	}
	if store.docs[0].Kind != sink.EmbedKindLead {
		t.Fatalf("kind=%q", store.docs[0].Kind)
	}
	if store.docs[0].TS.IsZero() {
		t.Fatal("expected ts")
	}
	_ = time.Now()
}
