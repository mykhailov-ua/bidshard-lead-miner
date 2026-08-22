package sink

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/semaphore"
)

// WarmAnalysisFailure is one lead row written to the warm-analysis DLQ.
type WarmAnalysisFailure struct {
	HashID   string
	Source   string
	Error    string
	Attempts int
}

// WarmAnalysisDLQWriter persists exhausted warm-path batch failures.
type WarmAnalysisDLQWriter interface {
	InsertWarmAnalysisFailures(ctx context.Context, failures []WarmAnalysisFailure) error
}

// WarmAnalysisDLQ is an append-only audit log for exhausted warm-path retries (one row per lead per batch).
type WarmAnalysisDLQ struct {
	coll       *mongo.Collection
	writeSlots *semaphore.Weighted
}

type warmAnalysisDLQDoc struct {
	TS       time.Time `bson:"ts"`
	HashID   string    `bson:"hash_id"`
	Source   string    `bson:"source"`
	Error    string    `bson:"error"`
	Attempts int       `bson:"attempts"`
}

func ConnectWarmAnalysisDLQ(ctx context.Context, client *mongo.Client, dbName, collection string, writeSlots int) (*WarmAnalysisDLQ, error) {
	if client == nil {
		return nil, errors.New("mongo client required")
	}
	if collection == "" {
		return nil, errors.New("warm analysis dlq collection required")
	}
	coll := client.Database(dbName).Collection(collection)
	dlq := &WarmAnalysisDLQ{
		coll:       coll,
		writeSlots: newWriteSlots(writeSlots),
	}
	if err := dlq.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return dlq, nil
}

func (d *WarmAnalysisDLQ) ensureIndexes(ctx context.Context) error {
	_, err := d.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "ts", Value: -1}}},
		{Keys: bson.D{{Key: "hash_id", Value: 1}}},
	})
	return err
}

func (d *WarmAnalysisDLQ) InsertWarmAnalysisFailures(ctx context.Context, failures []WarmAnalysisFailure) error {
	if d == nil || len(failures) == 0 {
		return nil
	}
	if err := d.writeSlots.Acquire(ctx, 1); err != nil {
		return err
	}
	defer d.writeSlots.Release(1)

	now := time.Now().UTC()
	docs := make([]interface{}, 0, len(failures))
	for _, f := range failures {
		if f.HashID == "" {
			continue
		}
		docs = append(docs, warmAnalysisDLQDoc{
			TS:       now,
			HashID:   f.HashID,
			Source:   f.Source,
			Error:    f.Error,
			Attempts: f.Attempts,
		})
	}
	if len(docs) == 0 {
		return nil
	}
	_, err := d.coll.InsertMany(ctx, docs)
	return err
}
