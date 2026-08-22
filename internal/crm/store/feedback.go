package store

import (
	"context"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WebhookFeedbackDoc records leads marked spam after CRM webhook delivery.
type WebhookFeedbackDoc struct {
	TS       time.Time `bson:"ts"`
	HashID   string    `bson:"hash_id"`
	Source   string    `bson:"source"`
	Priority string    `bson:"priority"`
	Score    int       `bson:"score"`
	HeatTier string    `bson:"heat_tier,omitempty"`
	Snippet  string    `bson:"snippet,omitempty"`
}

func (s *LeadStore) RecordWebhookSpamFeedback(ctx context.Context, lead sink.LeadDoc) error {
	if s == nil || s.webhookFeedback == nil || lead.HashID == "" {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()
	_, err := s.webhookFeedback.InsertOne(writeCtx, WebhookFeedbackDoc{
		TS:       time.Now().UTC(),
		HashID:   lead.HashID,
		Source:   lead.Source,
		Priority: lead.Priority,
		Score:    lead.Score,
		HeatTier: lead.HeatTier,
		Snippet:  lead.Snippet,
	})
	return err
}

func (s *LeadStore) ListWebhookSpamFeedback(ctx context.Context, since time.Time, limit int) ([]WebhookFeedbackDoc, error) {
	if s == nil || s.webhookFeedback == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()
	cur, err := s.webhookFeedback.Find(queryCtx,
		bson.M{"ts": bson.M{"$gte": since}},
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()
	var docs []WebhookFeedbackDoc
	return docs, cur.All(queryCtx, &docs)
}
