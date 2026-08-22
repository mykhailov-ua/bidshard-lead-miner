package sink

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GeoAuditDoc struct {
	TS           time.Time `bson:"ts" json:"ts"`
	AcceptSample int       `bson:"accept_sample" json:"accept_sample"`
	JunkSample   int       `bson:"junk_sample" json:"junk_sample"`
	AcceptSlips  int       `bson:"accept_slips" json:"accept_slips"`
	JunkMisses   int       `bson:"junk_misses" json:"junk_misses"`
	SlipRate     float64   `bson:"slip_rate" json:"slip_rate"`
	Summary      string    `bson:"summary,omitempty" json:"summary,omitempty"`
}

type GeoAuditStore struct {
	coll *mongo.Collection
}

func ConnectGeoAuditStore(ctx context.Context, client *mongo.Client, dbName, collection string) (*GeoAuditStore, error) {
	if client == nil {
		return nil, errors.New("mongo client required")
	}
	if collection == "" {
		collection = "geo_audit_reports"
	}
	coll := client.Database(dbName).Collection(collection)
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "ts", Value: -1}},
	})
	if err != nil {
		return nil, err
	}
	return &GeoAuditStore{coll: coll}, nil
}

func (s *GeoAuditStore) Insert(ctx context.Context, doc GeoAuditDoc) error {
	if s == nil {
		return nil
	}
	if doc.TS.IsZero() {
		doc.TS = time.Now().UTC()
	}
	_, err := s.coll.InsertOne(ctx, doc)
	return err
}

func (s *GeoAuditStore) Latest(ctx context.Context) (GeoAuditDoc, bool, error) {
	if s == nil {
		return GeoAuditDoc{}, false, nil
	}
	var doc GeoAuditDoc
	err := s.coll.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "ts", Value: -1}})).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return GeoAuditDoc{}, false, nil
	}
	if err != nil {
		return GeoAuditDoc{}, false, err
	}
	return doc, true, nil
}
