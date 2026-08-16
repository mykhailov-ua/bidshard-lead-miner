package sink

import (
	"context"
	"errors"
	"log/slog"

	"github.com/bidshard/parser/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/semaphore"
)

type MongoStore struct {
	coll       *mongo.Collection
	writeSlots *semaphore.Weighted
}

func ConnectMongo(ctx context.Context, uri, dbName, collection string, writeSlots int) (*MongoStore, error) {
	if uri == "" {
		return nil, errors.New("mongo uri required")
	}
	if dbName == "" {
		dbName = "parser"
	}
	if collection == "" {
		collection = "leads"
	}
	if writeSlots <= 0 {
		writeSlots = 8
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	store := &MongoStore{
		coll:       client.Database(dbName).Collection(collection),
		writeSlots: semaphore.NewWeighted(int64(writeSlots)),
	}
	if err := store.ensureIndexes(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return store, nil
}

func (s *MongoStore) ensureIndexes(ctx context.Context) error {
	_, err := s.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "hash_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "ts", Value: -1}}},
		{Keys: bson.D{{Key: "source", Value: 1}}},
	})
	return err
}

func (s *MongoStore) Exists(ctx context.Context, hashID string) (bool, error) {
	err := s.coll.FindOne(ctx, bson.M{"hash_id": hashID}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

func (s *MongoStore) Upsert(ctx context.Context, lead model.Lead) error {
	return s.upsertDoc(ctx, ToLeadDoc(lead))
}

func (s *MongoStore) BulkUpsert(ctx context.Context, leads []model.Lead) error {
	if len(leads) == 0 {
		return nil
	}
	if err := s.writeSlots.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.writeSlots.Release(1)

	models := make([]mongo.WriteModel, 0, len(leads))
	for _, lead := range leads {
		doc := ToLeadDoc(lead)
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"hash_id": doc.HashID}).
			SetUpdate(bson.M{"$setOnInsert": doc}).
			SetUpsert(true))
	}
	_, err := s.coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	if err != nil && !onlyDuplicateKeys(err) {
		return err
	}
	return nil
}

func (s *MongoStore) upsertDoc(ctx context.Context, doc LeadDoc) error {
	if err := s.writeSlots.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.writeSlots.Release(1)

	_, err := s.coll.UpdateOne(ctx,
		bson.M{"hash_id": doc.HashID},
		bson.M{"$setOnInsert": doc},
		options.Update().SetUpsert(true),
	)
	if err != nil && IsDuplicateKey(err) {
		return nil
	}
	return err
}

func OpenStore(ctx context.Context, uri, dbName, collection string, writeSlots int, exportPath string) Store {
	var stores []Store

	if uri != "" {
		if mongoStore, err := ConnectMongo(ctx, uri, dbName, collection, writeSlots); err != nil {
			slog.Warn("mongo connect failed", "error", err)
		} else {
			slog.Info("mongo sink connected", "db", dbName, "collection", collection, "write_slots", writeSlots)
			stores = append(stores, mongoStore)
		}
	}

	if exportPath != "" {
		fileSink, err := NewJSONFileSink(exportPath)
		if err != nil {
			slog.Warn("json export open failed", "path", exportPath, "error", err)
		} else {
			slog.Info("json export enabled", "path", exportPath)
			stores = append(stores, fileSink)
		}
	}

	switch len(stores) {
	case 0:
		slog.Debug("no sink configured, using stub store")
		return NewStubStore()
	case 1:
		return stores[0]
	default:
		return NewCompositeStore(stores...)
	}
}
