package store

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultSearchMaxRows = 20

type SearchResult struct {
	Leads     []sink.LeadDoc `json:"leads"`
	Truncated bool           `json:"truncated"`
}

func (s *LeadStore) Search(ctx context.Context, query string, limit int64, timeout time.Duration) (SearchResult, error) {
	if s == nil || s.leads == nil {
		return SearchResult{}, fmt.Errorf("lead store not initialized")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResult{}, fmt.Errorf("search query empty")
	}
	if limit <= 0 || limit > s.searchMaxRows {
		limit = s.searchMaxRows
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	pattern := regexp.QuoteMeta(query)
	filter := bson.M{
		"$or": bson.A{
			bson.M{"snippet": bson.M{"$regex": pattern, "$options": "i"}},
			bson.M{"source": bson.M{"$regex": pattern, "$options": "i"}},
		},
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Fetch limit+1 rows so callers can set Truncated without a separate count query.
	cur, err := s.leads.Find(queryCtx, filter,
		options.Find().
			SetSort(bson.D{{Key: "score", Value: -1}, {Key: "hash_id", Value: 1}}).
			SetLimit(limit+1).
			SetProjection(leadCardProjection()),
	)
	if err != nil {
		return SearchResult{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	docs := make([]sink.LeadDoc, 0, limit)
	for cur.Next(queryCtx) {
		var doc sink.LeadDoc
		if err := cur.Decode(&doc); err != nil {
			return SearchResult{}, err
		}
		docs = append(docs, doc)
	}
	if err := cur.Err(); err != nil {
		return SearchResult{}, err
	}

	truncated := int64(len(docs)) > limit
	if truncated {
		docs = docs[:limit]
	}
	return SearchResult{Leads: docs, Truncated: truncated}, nil
}
