package store

import (
	"context"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
)

// RecordWebhookSpamFeedback persists CRM spam marks for monthly webhook audit reports.
func (s *LeadStore) RecordWebhookSpamFeedback(ctx context.Context, lead sink.LeadDoc) error {
	if s == nil || s.webhookFeedback == nil {
		return nil
	}
	hashID := lead.HashID
	if hashID == "" {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()
	_, err := s.webhookFeedback.InsertOne(writeCtx, bson.M{
		"ts":        time.Now().UTC(),
		"hash_id":   hashID,
		"source":    lead.Source,
		"priority":  lead.Priority,
		"score":     lead.Score,
		"heat_tier": lead.HeatTier,
		"snippet":   lead.Snippet,
		"kind":      "spam",
	})
	return err
}
