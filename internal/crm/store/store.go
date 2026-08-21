package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultListLimit = 10
	maxListLimit     = 50
)

var ErrNotFound = errors.New("lead not found")

type LeadStore struct {
	leads         *mongo.Collection
	entities      *mongo.Collection
	sourceStats   *mongo.Collection
	keywordStats  *mongo.Collection
	crmBoosts     *mongo.Collection
	leadNotes     *mongo.Collection
	leadMeta      *mongo.Collection
	queryTimeout  time.Duration
	writeTimeout  time.Duration
	statsTimeout  time.Duration
	exportMaxRows int64
	searchMaxRows int64
}

func New(client *mongo.Client, opts Options) *LeadStore {
	if opts.QueryTimeout <= 0 {
		opts.QueryTimeout = 5 * time.Second
	}
	if opts.WriteTimeout <= 0 {
		opts.WriteTimeout = 3 * time.Second
	}
	db := client.Database(opts.DBName)
	s := &LeadStore{
		leads:         db.Collection(opts.LeadsCollection),
		queryTimeout:  opts.QueryTimeout,
		writeTimeout:  opts.WriteTimeout,
		statsTimeout:  opts.statsTimeout(),
		exportMaxRows: opts.exportMaxRows(),
		searchMaxRows: opts.searchMaxRows(),
	}
	if opts.EntityCollection != "" {
		s.entities = db.Collection(opts.EntityCollection)
	}
	if opts.SourceStatsCollection != "" {
		s.sourceStats = db.Collection(opts.SourceStatsCollection)
	}
	if opts.KeywordStatsCollection != "" {
		s.keywordStats = db.Collection(opts.KeywordStatsCollection)
	}
	if opts.CrmBoostCollection != "" {
		s.crmBoosts = db.Collection(opts.CrmBoostCollection)
	}
	if opts.LeadNotesCollection != "" {
		s.leadNotes = db.Collection(opts.LeadNotesCollection)
	}
	if opts.LeadCrmMetaCollection != "" {
		s.leadMeta = db.Collection(opts.LeadCrmMetaCollection)
	}
	return s
}

func (s *LeadStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	if s == nil || s.leads == nil {
		return fmt.Errorf("lead store not initialized")
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return fmt.Errorf("hash_id empty")
	}
	resolved, err := s.ResolveHashID(ctx, hashID)
	if err != nil {
		return err
	}
	hashID = resolved
	normalized, err := NormalizeStatus(status)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	res, err := s.leads.UpdateOne(writeCtx,
		bson.M{"hash_id": hashID},
		bson.M{"$set": bson.M{
			"status":    normalized,
			"status_at": time.Now().UTC(),
		}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

type ListFilter struct {
	Status       string
	SourcePrefix string
	ScoreMax     int
	Limit        int64
	Cursor       *ListCursor
	InboxOnly    bool   // exclude pending analysis and parser geo/icp rejects
	Sort         string // heat (default inbox) or score
}

type ListCursor struct {
	Score  int
	HashID string
}

type ListResult struct {
	Leads      []sink.LeadDoc `json:"leads"`
	NextCursor *ListCursor    `json:"next_cursor,omitempty"`
}

func (f ListFilter) clampLimit() int64 {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	return limit
}

func (f ListFilter) sortSpec() bson.D {
	if strings.EqualFold(strings.TrimSpace(f.Sort), "score") {
		return bson.D{
			{Key: "score", Value: -1},
			{Key: "hash_id", Value: 1},
		}
	}
	return bson.D{
		{Key: "entity_heat", Value: -1},
		{Key: "score", Value: -1},
		{Key: "hash_id", Value: 1},
	}
}

func (f ListFilter) matchQuery() bson.M {
	q := bson.M{}
	if f.InboxOnly {
		applyInboxFilter(q, strings.TrimSpace(f.Status))
	} else if status := f.Status; status != "" {
		q["status"] = status
	}
	if prefix := strings.TrimSpace(f.SourcePrefix); prefix != "" {
		q["source"] = bson.M{"$regex": "^" + regexp.QuoteMeta(prefix)}
	}
	if f.ScoreMax > 0 {
		q["score"] = bson.M{"$lte": f.ScoreMax}
	}
	if f.Cursor != nil {
		score := f.Cursor.Score
		hashID := f.Cursor.HashID
		q["$or"] = bson.A{
			bson.M{"score": bson.M{"$lt": score}},
			bson.M{"score": score, "hash_id": bson.M{"$gt": hashID}},
		}
	}
	return q
}

func (s *LeadStore) List(ctx context.Context, filter ListFilter) (ListResult, error) {
	if s == nil || s.leads == nil {
		return ListResult{}, fmt.Errorf("lead store not initialized")
	}

	limit := filter.clampLimit()
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	opts := options.Find().
		SetSort(filter.sortSpec()).
		// limit+1 detects whether a next page exists without CountDocuments.
		SetLimit(limit + 1).
		SetProjection(leadCardProjection())

	cur, err := s.leads.Find(queryCtx, filter.matchQuery(), opts)
	if err != nil {
		return ListResult{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	docs := make([]sink.LeadDoc, 0, limit)
	for cur.Next(queryCtx) {
		var doc sink.LeadDoc
		if err := cur.Decode(&doc); err != nil {
			return ListResult{}, err
		}
		docs = append(docs, doc)
	}
	if err := cur.Err(); err != nil {
		return ListResult{}, err
	}

	var next *ListCursor
	if int64(len(docs)) > limit {
		last := docs[limit-1]
		next = &ListCursor{Score: last.Score, HashID: last.HashID}
		docs = docs[:limit]
	}

	return ListResult{Leads: docs, NextCursor: next}, nil
}

func (s *LeadStore) GetByHashID(ctx context.Context, hashID string) (sink.LeadDoc, error) {
	if s == nil || s.leads == nil {
		return sink.LeadDoc{}, fmt.Errorf("lead store not initialized")
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return sink.LeadDoc{}, fmt.Errorf("hash_id empty")
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	var doc sink.LeadDoc
	err := s.leads.FindOne(queryCtx, bson.M{"hash_id": hashID}, options.FindOne().SetProjection(leadCardProjection())).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return sink.LeadDoc{}, ErrNotFound
	}
	if err != nil {
		return sink.LeadDoc{}, err
	}
	return doc, nil
}

func leadCardProjection() bson.M {
	return bson.M{
		"hash_id":               1,
		"ts":                    1,
		"round_id":              1,
		"priority":              1,
		"score":                 1,
		"source":                1,
		"title":                 1,
		"contacts":              1,
		"matched":               1,
		"snippet":               1,
		"status":                1,
		"status_at":             1,
		"outreach_draft":        1,
		"outreach_channel":      1,
		"outreach_angle":        1,
		"pilot_qualified":       1,
		"geo_country":           1,
		"entity_id":             1,
		"entity_heat":           1,
		"heat_tier":             1,
		"entity_sighting_count": 1,
		"entity_source_count":   1,
	}
}
