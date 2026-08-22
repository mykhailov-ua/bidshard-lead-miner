package sink

import (
	"context"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListWarmEntitiesForLinkSuggest returns warm+ entities for domain-pair review.
func (s *EntityStore) ListWarmEntitiesForLinkSuggest(ctx context.Context, limit int) ([]entity.EntityDoc, error) {
	if s == nil || s.coll == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tiers := entity.HeatTiersAtOrAbove(entity.HeatTierWarm)
	filter := bson.M{"unified_pain": bson.M{"$ne": ""}}
	if len(tiers) > 0 {
		filter["heat_tier"] = bson.M{"$in": tiers}
	}

	cur, err := s.coll.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "heat_score", Value: -1}}).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []entity.EntityDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// AppendReviewSuggestion adds a pending graph review row (no auto-merge).
func (s *EntityStore) AppendReviewSuggestion(ctx context.Context, entityID string, suggestion entity.ReviewSuggestion) error {
	if s == nil || entityID == "" || suggestion.PeerEntityID == "" {
		return nil
	}
	if suggestion.Status == "" {
		suggestion.Status = "pending"
	}
	if suggestion.TS.IsZero() {
		suggestion.TS = time.Now().UTC()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := s.coll.UpdateOne(ctx,
		bson.M{"entity_id": entityID},
		bson.M{"$push": bson.M{"review_suggestions": suggestion}},
	)
	return err
}
