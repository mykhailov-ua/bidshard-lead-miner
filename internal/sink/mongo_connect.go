package sink

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

func connectIndexedCollection(
	ctx context.Context,
	client *mongo.Client,
	dbName, collection, defaultCollection string,
	indexes []mongo.IndexModel,
) (*mongo.Collection, error) {
	if client == nil {
		return nil, errors.New("mongo client required")
	}
	if dbName == "" {
		dbName = "parser"
	}
	if collection == "" {
		collection = defaultCollection
	}
	coll := client.Database(dbName).Collection(collection)
	if len(indexes) == 0 {
		return coll, nil
	}
	if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
		return nil, err
	}
	return coll, nil
}

func connectIndexedCollectionOne(
	ctx context.Context,
	client *mongo.Client,
	dbName, collection, defaultCollection string,
	index mongo.IndexModel,
) (*mongo.Collection, error) {
	return connectIndexedCollection(ctx, client, dbName, collection, defaultCollection, []mongo.IndexModel{index})
}

func connectMongoStore(
	ctx context.Context,
	client *mongo.Client,
	dbName, collection string,
	writeSlots int,
	ensureIndexes bool,
) (*MongoStore, error) {
	if client == nil {
		return nil, errors.New("mongo client required")
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
	store := &MongoStore{
		coll:       client.Database(dbName).Collection(collection),
		writeSlots: newWriteSlots(writeSlots),
	}
	if ensureIndexes {
		if err := store.ensureIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return store, nil
}
