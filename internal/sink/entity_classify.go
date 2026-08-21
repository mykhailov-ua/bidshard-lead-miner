package sink

import (
	"context"
	"errors"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *EntityStore) GetEntity(ctx context.Context, entityID string) (entity.EntityDoc, bool, error) {
	if s == nil || entityID == "" {
		return entity.EntityDoc{}, false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var doc entity.EntityDoc
	err := s.coll.FindOne(ctx, bson.M{"entity_id": entityID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entity.EntityDoc{}, false, nil
	}
	if err != nil {
		return entity.EntityDoc{}, false, err
	}
	return doc, true, nil
}

func (s *EntityStore) PatchEntityClassification(ctx context.Context, entityID string, patch entity.EntityClassificationPatch) error {
	if s == nil || entityID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	set := bson.M{
		"unified_pain":         patch.UnifiedPain,
		"actor_confidence":     patch.ActorConfidence,
		"buyer_intent":         patch.BuyerIntent,
		"needs_review":         patch.NeedsReview,
		"heat_tier":            patch.HeatTier,
		"heat_score":           patch.HeatScore,
		"entity_classified_at": patch.EntityClassifiedAt,
	}
	_, err := s.coll.UpdateOne(ctx,
		bson.M{"entity_id": entityID},
		bson.M{"$set": set},
		options.Update(),
	)
	return err
}
