package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetExplainCache returns a cached explain string when still fresh.
func (s *LeadStore) GetExplainCache(ctx context.Context, hashID string) (string, bool, error) {
	meta, err := s.GetMeta(ctx, hashID)
	if err != nil {
		return "", false, err
	}
	if meta.ExplainSummary == "" || meta.ExplainCachedAt.IsZero() {
		return "", false, nil
	}
	if !meta.ExplainExpiresAt.IsZero() {
		if time.Now().UTC().After(meta.ExplainExpiresAt) {
			return "", false, nil
		}
	} else if time.Since(meta.ExplainCachedAt) > 24*time.Hour {
		return "", false, nil
	}
	return meta.ExplainSummary, true, nil
}

// SetExplainCache stores explain summary for 24h CRM inbox reuse.
func (s *LeadStore) SetExplainCache(ctx context.Context, hashID, summary string, ttl time.Duration) error {
	if s == nil || s.leadMeta == nil {
		return nil
	}
	hashID, err := s.resolveExistingHash(ctx, hashID)
	if err != nil {
		return err
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fmt.Errorf("explain summary empty")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()
	_, err = s.leadMeta.UpdateOne(writeCtx,
		bson.M{"hash_id": hashID},
		bson.M{"$set": bson.M{
			"explain_summary":    summary,
			"explain_cached_at":  now,
			"explain_expires_at": expires,
			"updated_at":         now,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}
