package sink

import (
	"context"
	"errors"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EntityHeatPatch updates canonical lead fields after entity heat recompute.
type EntityHeatPatch struct {
	HeatScore     float64
	HeatTier      string
	SightingCount int
	SourceCount   int
	Boost         int
}

// EntityHeatPatcher updates a stored lead when entity heat crosses hot bands.
type EntityHeatPatcher interface {
	ApplyEntityHeat(ctx context.Context, hashID string, patch EntityHeatPatch) error
}

func (s *MongoStore) ApplyEntityHeat(ctx context.Context, hashID string, patch EntityHeatPatch) error {
	if s == nil || hashID == "" {
		return nil
	}
	if err := s.writeSlots.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.writeSlots.Release(1)

	var doc LeadDoc
	err := s.coll.FindOne(ctx, bson.M{"hash_id": hashID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}
		return err
	}

	set := bson.M{
		"entity_heat":           patch.HeatScore,
		"heat_tier":             patch.HeatTier,
		"entity_sighting_count": patch.SightingCount,
		"entity_source_count":   patch.SourceCount,
	}
	if entity.HeatTierRank(patch.HeatTier) >= entity.HeatTierRank(entity.HeatTierHot) {
		boost := patch.Boost
		if boost <= 0 {
			boost = entity.DefaultCrossSourceBoost
		}
		set["hot"] = true
		set["score"] = doc.Score + boost
		if doc.Priority == "Medium" {
			set["priority"] = "High"
		}
		tags := append([]string(nil), doc.Tags...)
		found := false
		for _, tag := range tags {
			if tag == entity.TagCrossSourceHot {
				found = true
				break
			}
		}
		if !found {
			tags = append(tags, entity.TagCrossSourceHot)
		}
		set["tags"] = tags
	}

	_, err = s.coll.UpdateOne(ctx,
		bson.M{"hash_id": hashID},
		bson.M{"$set": set},
		options.Update(),
	)
	return err
}
