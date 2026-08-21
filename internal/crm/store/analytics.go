package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type SourceCount struct {
	Source string
	Count  int64
}

type StatsOverview struct {
	Count24h     int64
	Count7d      int64
	StatusFunnel []StatusCount
	TopSources7d []SourceCount
}

type SourceStatsRow struct {
	Source   string
	Accepted int64
	Junk     int64
	Fallback bool
}

type KeywordStatsRow struct {
	KeywordID string
	Accepted  int64
	Junk      int64
}

func (s *LeadStore) StatsOverview(ctx context.Context) (StatsOverview, error) {
	if s == nil || s.leads == nil {
		return StatsOverview{}, fmt.Errorf("lead store not initialized")
	}

	now := time.Now().UTC()
	since24h := now.Add(-24 * time.Hour)
	since7d := now.Add(-7 * 24 * time.Hour)

	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$facet", Value: bson.M{
			"status_funnel": bson.A{
				bson.M{"$group": bson.M{"_id": "$status", "count": bson.M{"$sum": 1}}},
				bson.M{"$sort": bson.M{"count": -1}},
			},
			"count_24h": bson.A{
				bson.M{"$match": bson.M{"ts": bson.M{"$gte": since24h}}},
				bson.M{"$count": "n"},
			},
			"count_7d": bson.A{
				bson.M{"$match": bson.M{"ts": bson.M{"$gte": since7d}}},
				bson.M{"$count": "n"},
			},
			"top_sources_7d": bson.A{
				bson.M{"$match": bson.M{"ts": bson.M{"$gte": since7d}}},
				bson.M{"$group": bson.M{"_id": "$source", "count": bson.M{"$sum": 1}}},
				bson.M{"$sort": bson.M{"count": -1}},
				bson.M{"$limit": maxTopSources},
			},
		}}},
	}

	cur, err := s.leads.Aggregate(queryCtx, pipeline)
	if err != nil {
		return StatsOverview{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var rows []bson.M
	if err := cur.All(queryCtx, &rows); err != nil {
		return StatsOverview{}, err
	}
	if len(rows) == 0 {
		return StatsOverview{}, nil
	}

	out := StatsOverview{}
	root := rows[0]

	out.StatusFunnel = decodeStatusFunnel(root["status_funnel"])
	out.Count24h = decodeCount(root["count_24h"])
	out.Count7d = decodeCount(root["count_7d"])
	out.TopSources7d = decodeTopSources(root["top_sources_7d"])
	return out, nil
}

func (s *LeadStore) StatsSources(ctx context.Context) ([]SourceStatsRow, error) {
	if s == nil || s.leads == nil {
		return nil, fmt.Errorf("lead store not initialized")
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()

	if s.sourceStats != nil {
		cur, err := s.sourceStats.Find(queryCtx, bson.M{},
			options.Find().SetSort(bson.D{{Key: "accepted", Value: -1}}).SetLimit(maxTopSources))
		if err != nil {
			return nil, err
		}
		defer func() { _ = cur.Close(queryCtx) }()

		var docs []sink.SourceStatsDoc
		if err := cur.All(queryCtx, &docs); err != nil {
			return nil, err
		}
		if len(docs) > 0 {
			rows := make([]SourceStatsRow, 0, len(docs))
			for _, doc := range docs {
				rows = append(rows, SourceStatsRow{
					Source:   doc.Source,
					Accepted: int64(doc.Accepted),
					Junk:     int64(doc.Junk),
				})
			}
			return rows, nil
		}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{"_id": "$source", "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
		{{Key: "$limit", Value: maxTopSources}},
	}
	cur, err := s.leads.Aggregate(queryCtx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	type aggRow struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	var agg []aggRow
	if err := cur.All(queryCtx, &agg); err != nil {
		return nil, err
	}
	rows := make([]SourceStatsRow, 0, len(agg))
	for _, row := range agg {
		rows = append(rows, SourceStatsRow{
			Source:   row.ID,
			Accepted: row.Count,
			Fallback: true,
		})
	}
	return rows, nil
}

func (s *LeadStore) StatsKeywords(ctx context.Context) ([]KeywordStatsRow, error) {
	if s == nil || s.keywordStats == nil {
		return nil, fmt.Errorf("keyword stats collection not configured")
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()

	cur, err := s.keywordStats.Find(queryCtx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "accepted_count", Value: -1}}).SetLimit(maxKeywordStats))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var docs []sink.KeywordStatDoc
	if err := cur.All(queryCtx, &docs); err != nil {
		return nil, err
	}
	rows := make([]KeywordStatsRow, 0, len(docs))
	for _, doc := range docs {
		rows = append(rows, KeywordStatsRow{
			KeywordID: doc.KeywordID,
			Accepted:  int64(doc.AcceptedCount),
			Junk:      int64(doc.JunkCount),
		})
	}
	return rows, nil
}

func (s *LeadStore) ListBoosts(ctx context.Context, limit int64) ([]sink.CrmBoostDoc, error) {
	if s == nil || s.crmBoosts == nil {
		return nil, fmt.Errorf("crm boosts collection not configured")
	}
	if limit <= 0 || limit > maxBoostRows {
		limit = maxBoostRows
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()

	cur, err := s.crmBoosts.Find(queryCtx,
		bson.M{"status": "pending"},
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var docs []sink.CrmBoostDoc
	if err := cur.All(queryCtx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func decodeStatusFunnel(raw any) []StatusCount {
	items, ok := asBsonArray(raw)
	if !ok {
		return nil
	}
	out := make([]StatusCount, 0, len(items))
	for _, item := range items {
		doc, ok := asBsonMap(item)
		if !ok {
			continue
		}
		status, _ := doc["_id"].(string)
		if status == "" {
			status = "unknown"
		}
		count := int64(0)
		switch v := doc["count"].(type) {
		case int32:
			count = int64(v)
		case int64:
			count = v
		case float64:
			count = int64(v)
		}
		out = append(out, StatusCount{Status: status, Count: count})
	}
	return out
}

func decodeCount(raw any) int64 {
	items, ok := asBsonArray(raw)
	if !ok || len(items) == 0 {
		return 0
	}
	doc, ok := asBsonMap(items[0])
	if !ok {
		return 0
	}
	switch v := doc["n"].(type) {
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func decodeTopSources(raw any) []SourceCount {
	items, ok := asBsonArray(raw)
	if !ok {
		return nil
	}
	out := make([]SourceCount, 0, len(items))
	for _, item := range items {
		doc, ok := asBsonMap(item)
		if !ok {
			continue
		}
		source, _ := doc["_id"].(string)
		count := int64(0)
		switch v := doc["count"].(type) {
		case int32:
			count = int64(v)
		case int64:
			count = v
		case float64:
			count = int64(v)
		}
		out = append(out, SourceCount{Source: source, Count: count})
	}
	return out
}

func asBsonArray(raw any) (bson.A, bool) {
	switch v := raw.(type) {
	case bson.A:
		return v, true
	case []interface{}:
		return bson.A(v), true
	default:
		return nil, false
	}
}

func asBsonMap(item any) (bson.M, bool) {
	switch v := item.(type) {
	case bson.M:
		return v, true
	case map[string]interface{}:
		return bson.M(v), true
	default:
		return nil, false
	}
}
