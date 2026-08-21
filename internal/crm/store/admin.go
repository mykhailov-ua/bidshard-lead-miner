package store

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Cap each purge/delete pass so a typo in CLI filters cannot wipe the whole DB silently.
const maxAdminDeleteBatch = 10000

// DeleteFilter selects leads to remove. Set All for full purge (requires explicit confirm elsewhere).
type DeleteFilter struct {
	HashID       string
	Status       string
	SourcePrefix string
	ScoreMax     int
	All          bool
}

type DeleteResult struct {
	Leads int64 `json:"leads"`
	Notes int64 `json:"notes"`
	Meta  int64 `json:"meta"`
}

type DBStats struct {
	TotalLeads int64         `json:"total_leads"`
	ByStatus   []StatusCount `json:"by_status"`
}

func (f DeleteFilter) Validate() error {
	if f.All {
		return nil
	}
	if strings.TrimSpace(f.HashID) != "" {
		return nil
	}
	if strings.TrimSpace(f.Status) != "" {
		return nil
	}
	if strings.TrimSpace(f.SourcePrefix) != "" {
		return nil
	}
	if f.ScoreMax > 0 {
		return nil
	}
	return fmt.Errorf("delete filter empty: set hash, status, source prefix, score max, or --all")
}

func (f DeleteFilter) matchQuery() bson.M {
	if f.All {
		return bson.M{}
	}
	q := bson.M{}
	if hashID := strings.TrimSpace(f.HashID); hashID != "" {
		q["hash_id"] = hashID
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		q["status"] = status
	}
	if prefix := strings.TrimSpace(f.SourcePrefix); prefix != "" {
		q["source"] = bson.M{"$regex": "^" + bsonRegexEscape(prefix)}
	}
	if f.ScoreMax > 0 {
		q["score"] = bson.M{"$lte": f.ScoreMax}
	}
	return q
}

func bsonRegexEscape(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`, `.`, `\.`, `+`, `\+`, `*`, `\*`, `?`, `\?`,
		`^`, `\^`, `$`, `\$`, `{`, `\{`, `}`, `\}`, `(`, `\(`,
		`)`, `\)`, `[`, `\[`, `]`, `\]`, `|`, `\|`,
	)
	return replacer.Replace(s)
}

func (s *LeadStore) DBStats(ctx context.Context) (DBStats, error) {
	if s == nil || s.leads == nil {
		return DBStats{}, fmt.Errorf("lead store not initialized")
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()

	total, err := s.leads.CountDocuments(queryCtx, bson.M{})
	if err != nil {
		return DBStats{}, err
	}

	cur, err := s.leads.Aggregate(queryCtx, mongoPipelineGroupByStatus())
	if err != nil {
		return DBStats{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var rows []bson.M
	if err := cur.All(queryCtx, &rows); err != nil {
		return DBStats{}, err
	}

	return DBStats{
		TotalLeads: total,
		ByStatus:   decodeStatusFunnel(rows),
	}, nil
}

func mongoPipelineGroupByStatus() bson.A {
	return bson.A{
		bson.M{"$group": bson.M{"_id": "$status", "count": bson.M{"$sum": 1}}},
		bson.M{"$sort": bson.M{"count": -1}},
	}
}

func (s *LeadStore) DeleteLeads(ctx context.Context, filter DeleteFilter) (DeleteResult, error) {
	if s == nil || s.leads == nil {
		return DeleteResult{}, fmt.Errorf("lead store not initialized")
	}
	if err := filter.Validate(); err != nil {
		return DeleteResult{}, err
	}
	if hashID := strings.TrimSpace(filter.HashID); hashID != "" {
		resolved, err := s.ResolveHashID(ctx, hashID)
		if err != nil {
			return DeleteResult{}, err
		}
		filter.HashID = resolved
	}

	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout*10)
	defer cancel()

	q := filter.matchQuery()
	hashIDs, err := s.collectHashIDs(writeCtx, q, maxAdminDeleteBatch)
	if err != nil {
		return DeleteResult{}, err
	}
	if len(hashIDs) == 0 {
		return DeleteResult{}, nil
	}

	in := bson.M{"hash_id": bson.M{"$in": hashIDs}}
	leadRes, err := s.leads.DeleteMany(writeCtx, in)
	if err != nil {
		return DeleteResult{}, err
	}

	out := DeleteResult{Leads: leadRes.DeletedCount}
	if s.leadNotes != nil {
		nRes, err := s.leadNotes.DeleteMany(writeCtx, in)
		if err != nil {
			return out, fmt.Errorf("delete notes: %w", err)
		}
		out.Notes = nRes.DeletedCount
	}
	if s.leadMeta != nil {
		mRes, err := s.leadMeta.DeleteMany(writeCtx, in)
		if err != nil {
			return out, fmt.Errorf("delete meta: %w", err)
		}
		out.Meta = mRes.DeletedCount
	}
	return out, nil
}

func (s *LeadStore) collectHashIDs(ctx context.Context, q bson.M, limit int) ([]string, error) {
	opts := optionsFindHashIDs(limit)
	cur, err := s.leads.Find(ctx, q, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	ids := make([]string, 0, 64)
	for cur.Next(ctx) {
		var row struct {
			HashID string `bson:"hash_id"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.HashID != "" {
			ids = append(ids, row.HashID)
		}
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	// Bulk filters must match fewer than maxAdminDeleteBatch unless deleting one hash_id.
	if len(ids) >= limit && !isSingleHashQuery(q) {
		return nil, fmt.Errorf("delete matched >= %d leads; narrow the filter", limit)
	}
	return ids, nil
}

func isSingleHashQuery(q bson.M) bool {
	hashID, ok := q["hash_id"].(string)
	return ok && strings.TrimSpace(hashID) != ""
}

func optionsFindHashIDs(limit int) *options.FindOptions {
	return options.Find().SetProjection(bson.M{"hash_id": 1}).SetLimit(int64(limit))
}
