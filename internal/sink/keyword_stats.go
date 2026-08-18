package sink

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type KeywordStatDoc struct {
	KeywordID     string    `bson:"keyword_id" json:"keyword_id"`
	AcceptedCount int       `bson:"accepted_count" json:"accepted_count"`
	JunkCount     int       `bson:"junk_count" json:"junk_count"`
	LastSeenAt    time.Time `bson:"last_seen_at" json:"last_seen_at"`
}

func (d KeywordStatDoc) TotalSamples() int {
	return d.AcceptedCount + d.JunkCount
}

func (d KeywordStatDoc) JunkRate() float64 {
	total := d.TotalSamples()
	if total == 0 {
		return 0.0
	}
	return float64(d.JunkCount) / float64(total)
}

func (d KeywordStatDoc) Precision() float64 {
	return 1.0 - d.JunkRate()
}

// Recommendation returns weight/enabled recommendation based on P1-2 rules.
// Rule: if total samples >= 20 and junk_rate > 30% (0.3), recommend enabled: false.
func (d KeywordStatDoc) Recommendation(currentWeight int) (suggestedWeight int, enabled bool) {
	junkRate := d.JunkRate()
	total := d.TotalSamples()

	if total >= 20 && junkRate > 0.30 {
		return currentWeight, false
	}

	if junkRate > 0.20 {
		w := currentWeight - 5
		if w < 5 {
			w = 5
		}
		return w, true
	}
	if junkRate < 0.05 && total >= 10 {
		w := currentWeight + 5
		if w > 30 {
			w = 30
		}
		return w, true
	}

	return currentWeight, true
}

type KeywordStatsStore struct {
	coll *mongo.Collection
}

func ConnectKeywordStats(ctx context.Context, client *mongo.Client, dbName, collection string) (*KeywordStatsStore, error) {
	if dbName == "" {
		dbName = "parser"
	}
	if collection == "" {
		collection = "keyword_stats"
	}
	coll := client.Database(dbName).Collection(collection)
	store := &KeywordStatsStore{coll: coll}
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "keyword_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (s *KeywordStatsStore) RecordOutcome(ctx context.Context, keywordID string, isJunk bool) error {
	if keywordID == "" {
		return nil
	}
	incField := "accepted_count"
	if isJunk {
		incField = "junk_count"
	}

	_, err := s.coll.UpdateOne(ctx,
		bson.M{"keyword_id": keywordID},
		bson.M{
			"$inc": bson.M{incField: 1},
			"$set": bson.M{"last_seen_at": time.Now().UTC()},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *KeywordStatsStore) GetStats(ctx context.Context, keywordID string) (KeywordStatDoc, error) {
	var doc KeywordStatDoc
	err := s.coll.FindOne(ctx, bson.M{"keyword_id": keywordID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return KeywordStatDoc{KeywordID: keywordID}, nil
	}
	return doc, err
}
