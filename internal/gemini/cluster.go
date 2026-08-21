package gemini

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// LeadCluster detects semantic duplicates among accepted leads.
type LeadCluster struct {
	embedder  TextEmbedder
	store     leadEmbedStore
	threshold float64
}

type leadEmbedStore interface {
	RecentVectorsByKind(ctx context.Context, kind string, limit int) ([]sink.EmbeddingDoc, error)
	Upsert(ctx context.Context, doc sink.EmbeddingDoc) error
}

func NewLeadCluster(embedder TextEmbedder, store leadEmbedStore, threshold float64) *LeadCluster {
	if threshold <= 0 {
		threshold = 0.92
	}
	return &LeadCluster{
		embedder:  embedder,
		store:     store,
		threshold: threshold,
	}
}

// CheckDuplicate returns true when snippet matches a different accepted lead.
func (c *LeadCluster) CheckDuplicate(ctx context.Context, hashID, text string) (bool, string, error) {
	if c == nil || c.embedder == nil || c.store == nil {
		return false, "", nil
	}
	hashID = strings.TrimSpace(hashID)
	text = strings.TrimSpace(text)
	if hashID == "" || text == "" {
		return false, "", nil
	}

	vec, err := c.embedder.EmbedText(ctx, text)
	if err != nil {
		return false, "", err
	}

	recent, err := c.store.RecentVectorsByKind(ctx, sink.EmbedKindLead, 500)
	if err != nil {
		return false, "", err
	}
	// Linear scan of recent leads; same HashID skipped but near-duplicate text with different hash is not.
	for _, other := range recent {
		if other.HashID == hashID || other.HashID == "" {
			continue
		}
		if CosineSimilarity(vec, other.Vector) >= c.threshold {
			return true, other.HashID, nil
		}
	}
	return false, "", nil
}

// Record stores the accepted lead embedding for future clustering.
func (c *LeadCluster) Record(ctx context.Context, hashID, text string) error {
	if c == nil || c.embedder == nil || c.store == nil {
		return nil
	}
	hashID = strings.TrimSpace(hashID)
	text = strings.TrimSpace(text)
	if hashID == "" || text == "" {
		return nil
	}

	vec, err := c.embedder.EmbedText(ctx, text)
	if err != nil {
		return fmt.Errorf("lead cluster embed: %w", err)
	}
	return c.store.Upsert(ctx, sink.EmbeddingDoc{
		Key:    sink.LeadEmbedKey(hashID),
		HashID: hashID,
		Vector: vec,
		Kind:   sink.EmbedKindLead,
		TS:     time.Now().UTC(),
	})
}
