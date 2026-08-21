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

// EntityStore persists cross-source entity sightings in MongoDB.
type EntityStore struct {
	coll              *mongo.Collection
	CrossSourceWindow time.Duration
}

func ConnectEntityStore(ctx context.Context, client *mongo.Client, dbName, collection string) (*EntityStore, error) {
	coll, err := connectIndexedCollection(ctx, client, dbName, collection, "entities", []mongo.IndexModel{
		{Keys: bson.D{{Key: "entity_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "alias_keys", Value: 1}}},
		{Keys: bson.D{{Key: "last_seen", Value: -1}}},
	})
	if err != nil {
		return nil, err
	}
	return &EntityStore{coll: coll}, nil
}

func (s *EntityStore) RecordSighting(ctx context.Context, in entity.SightingInput) (entity.RecordResult, error) {
	if s == nil {
		return entity.RecordResult{}, nil
	}
	return entity.RecordSightingCore(ctx, in, s.CrossSourceWindow, entity.DefaultMergeMaxAttempts, s)
}

func (s *EntityStore) FindExisting(ctx context.Context, aliases []string, entityID string) (*entity.EntityDoc, error) {
	var filter bson.M
	switch {
	case len(aliases) > 0 && entityID != "":
		filter = bson.M{"$or": bson.A{
			bson.M{"entity_id": entityID},
			bson.M{"alias_keys": bson.M{"$in": aliases}},
		}}
	case entityID != "":
		filter = bson.M{"entity_id": entityID}
	case len(aliases) > 0:
		filter = bson.M{"alias_keys": bson.M{"$in": aliases}}
	default:
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var doc entity.EntityDoc
	err := s.coll.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "first_seen", Value: 1}})).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *EntityStore) InsertNew(ctx context.Context, doc entity.EntityDoc) (duplicate bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = s.coll.InsertOne(ctx, doc)
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return true, nil
	}
	return false, err
}

func (s *EntityStore) ReplaceMerged(ctx context.Context, doc entity.EntityDoc, generation int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := s.coll.ReplaceOne(ctx, bson.M{
		"entity_id":        doc.EntityID,
		"merge_generation": generation,
	}, doc)
	if err != nil {
		return false, err
	}
	return res.MatchedCount == 1, nil
}
