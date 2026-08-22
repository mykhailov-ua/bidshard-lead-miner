package sink

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WebhookFeedbackDoc struct {
	TS       time.Time `bson:"ts"`
	HashID   string    `bson:"hash_id"`
	Source   string    `bson:"source"`
	Priority string    `bson:"priority"`
	Score    int       `bson:"score"`
	HeatTier string    `bson:"heat_tier,omitempty"`
	Snippet  string    `bson:"snippet,omitempty"`
}

type WebhookFeedbackStore struct {
	coll *mongo.Collection
}

func ConnectWebhookFeedbackStore(ctx context.Context, client *mongo.Client, dbName, collection string) (*WebhookFeedbackStore, error) {
	if client == nil {
		return nil, errors.New("mongo client required")
	}
	if collection == "" {
		collection = "webhook_feedback"
	}
	coll := client.Database(dbName).Collection(collection)
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "ts", Value: -1}},
	})
	if err != nil {
		return nil, err
	}
	return &WebhookFeedbackStore{coll: coll}, nil
}

func (s *WebhookFeedbackStore) ListSince(ctx context.Context, since time.Time, limit int) ([]WebhookFeedbackDoc, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	cur, err := s.coll.Find(ctx,
		bson.M{"ts": bson.M{"$gte": since}},
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []WebhookFeedbackDoc
	return docs, cur.All(ctx, &docs)
}
