package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// OutcomeSourceCount is a flattened source/outcome count from leads.
type OutcomeSourceCount struct {
	Source  string
	Outcome string
	Count   int64
}

// AggregateOutcomeBySource groups non-empty lead outcomes by source.
func AggregateOutcomeBySource(ctx context.Context, leads *mongo.Collection, timeout time.Duration) ([]OutcomeSourceCount, error) {
	if leads == nil {
		return nil, nil
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"outcome": bson.M{"$exists": true, "$ne": ""}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"source": "$source", "outcome": "$outcome"},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}
	cur, err := leads.Aggregate(queryCtx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var rows []struct {
		ID struct {
			Source  string `bson:"source"`
			Outcome string `bson:"outcome"`
		} `bson:"_id"`
		Count int64 `bson:"count"`
	}
	if err := cur.All(queryCtx, &rows); err != nil {
		return nil, err
	}
	out := make([]OutcomeSourceCount, 0, len(rows))
	for _, row := range rows {
		if row.ID.Source == "" || row.ID.Outcome == "" || row.Count == 0 {
			continue
		}
		out = append(out, OutcomeSourceCount{
			Source:  row.ID.Source,
			Outcome: row.ID.Outcome,
			Count:   row.Count,
		})
	}
	return out, nil
}
