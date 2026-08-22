package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListStaleNewLeads returns status=new leads older than cutoff for ICP re-grade.
func (s *MongoStore) ListStaleNewLeads(ctx context.Context, olderThan time.Duration, limit int) ([]LeadDoc, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	filter := bson.M{
		"status": "new",
		"ts":     bson.M{"$lte": cutoff},
	}
	cur, err := s.coll.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "ts", Value: 1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []LeadDoc
	return docs, cur.All(ctx, &docs)
}

// ListHighLeadsSince returns High priority leads accepted since cutoff.
func (s *MongoStore) ListHighLeadsSince(ctx context.Context, since time.Time, limit int) ([]LeadDoc, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	filter := bson.M{
		"priority":     "High",
		"ts":           bson.M{"$gte": since},
		"duplicate_of": bson.M{"$in": bson.A{"", nil}},
	}
	cur, err := s.coll.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []LeadDoc
	return docs, cur.All(ctx, &docs)
}

// PatchDuplicateSuggest sets duplicate_suggest on a lead (human approve in CRM).
func (s *MongoStore) PatchDuplicateSuggest(ctx context.Context, hashID, suggestHash string) error {
	if s == nil || hashID == "" || suggestHash == "" {
		return nil
	}
	if err := s.writeSlots.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.writeSlots.Release(1)
	_, err := s.coll.UpdateOne(ctx,
		bson.M{"hash_id": hashID},
		bson.M{"$set": bson.M{"duplicate_suggest": suggestHash}},
	)
	return err
}

// SampleRandomAccepts returns random accepted leads since cutoff for geo audit.
func (s *MongoStore) SampleRandomAccepts(ctx context.Context, since time.Time, limit int) ([]LeadDoc, error) {
	if s == nil || limit <= 0 {
		return nil, nil
	}
	pipe := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"ts":              bson.M{"$gte": since},
			"analysis_status": bson.M{"$in": bson.A{"done", ""}},
			"status":          bson.M{"$nin": bson.A{"geo_rejected", "icp_rejected"}},
		}}},
		{{Key: "$sample", Value: bson.M{"size": limit}}},
	}
	cur, err := s.coll.Aggregate(ctx, pipe)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []LeadDoc
	return docs, cur.All(ctx, &docs)
}
