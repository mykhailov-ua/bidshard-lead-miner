package sink

import (
	"context"
	"errors"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CrossSourceHotPatcher updates a stored lead when cross-source hot triggers.
type CrossSourceHotPatcher interface {
	ApplyCrossSourceHot(ctx context.Context, hashID string, boost int) error
}

func (s *MongoStore) ApplyCrossSourceHot(ctx context.Context, hashID string, boost int) error {
	if s == nil || hashID == "" {
		return nil
	}
	if boost <= 0 {
		boost = entity.DefaultCrossSourceBoost
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
		"hot":   true,
		"score": doc.Score + boost,
	}
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

	_, err = s.coll.UpdateOne(ctx,
		bson.M{"hash_id": hashID},
		bson.M{"$set": set},
		options.Update(),
	)
	return err
}
