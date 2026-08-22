package sink

import (
	"context"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListUnifiedPainSamples returns distinct unified_pain values from warm+ entities.
func (s *EntityStore) ListUnifiedPainSamples(ctx context.Context, limit int) ([]string, error) {
	if s == nil || s.coll == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tiers := entity.HeatTiersAtOrAbove(entity.HeatTierWarm)
	filter := bson.M{
		"unified_pain": bson.M{"$ne": ""},
	}
	if len(tiers) > 0 {
		filter["heat_tier"] = bson.M{"$in": tiers}
	}

	cur, err := s.coll.Find(ctx, filter,
		options.Find().
			SetProjection(bson.M{"unified_pain": 1}).
			SetSort(bson.D{{Key: "heat_score", Value: -1}}).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	seen := make(map[string]struct{}, limit)
	out := make([]string, 0, limit)
	for cur.Next(ctx) {
		var row struct {
			UnifiedPain string `bson:"unified_pain"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		pain := strings.TrimSpace(row.UnifiedPain)
		if pain == "" {
			continue
		}
		key := strings.ToLower(pain)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pain)
	}
	return out, cur.Err()
}
