package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *LeadStore) ResolveBoost(ctx context.Context, id, status, leadHashID, why string) error {
	if s == nil || s.crmBoosts == nil {
		return fmt.Errorf("crm boosts collection not configured")
	}
	oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("invalid boost id: %w", err)
	}
	set := bson.M{
		"status":      status,
		"resolved_at": time.Now().UTC(),
	}
	if leadHashID != "" {
		set["lead_hash_id"] = leadHashID
	}
	if why != "" {
		set["outcome_why"] = why
	}
	res, err := s.crmBoosts.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("boost not found")
	}
	return nil
}

func (s *LeadStore) ListAllBoosts(ctx context.Context, status string, limit int64) ([]sink.CrmBoostDoc, error) {
	if s == nil || s.crmBoosts == nil {
		return nil, fmt.Errorf("crm boosts collection not configured")
	}
	if limit <= 0 || limit > maxBoostRows {
		limit = maxBoostRows
	}
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()
	cur, err := s.crmBoosts.Find(queryCtx, filter,
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()
	var docs []sink.CrmBoostDoc
	return docs, cur.All(queryCtx, &docs)
}
