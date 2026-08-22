package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StalePendingLister reads deferred leads stuck in analysis_status=pending.
type StalePendingLister interface {
	ListStalePendingLeads(ctx context.Context, olderThan time.Duration, limit int) ([]LeadDoc, error)
	CountPendingAnalysis(ctx context.Context) (int64, error)
}

func (s *MongoStore) ListStalePendingLeads(ctx context.Context, olderThan time.Duration, limit int) ([]LeadDoc, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 15
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	// Pending defer leads have no analysis_at; accept-time ts is the staleness clock.
	filter := bson.M{
		"analysis_status": "pending",
		"ts":              bson.M{"$lte": cutoff},
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "ts", Value: 1}}).
		SetLimit(int64(limit))
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []LeadDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (s *MongoStore) CountPendingAnalysis(ctx context.Context) (int64, error) {
	if s == nil {
		return 0, nil
	}
	return s.coll.CountDocuments(ctx, bson.M{"analysis_status": "pending"})
}

// FindHashIDByContactValue returns a lead hash_id when contacts.value matches hint.
func (s *MongoStore) FindHashIDByContactValue(ctx context.Context, value string) (string, error) {
	if s == nil || value == "" {
		return "", nil
	}
	var doc LeadDoc
	err := s.coll.FindOne(ctx, bson.M{"contacts.value": value}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", err
	}
	return doc.HashID, nil
}
