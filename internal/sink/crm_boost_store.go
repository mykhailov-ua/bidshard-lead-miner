package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CrmBoost statuses after worker or manual CRM ops.
const (
	CrmBoostPending   = "pending"
	CrmBoostPromoted  = "promoted"
	CrmBoostDismissed = "dismissed"
	CrmBoostMerged    = "merged"
)

func (s *CrmBoostStore) ListPending(ctx context.Context, limit int) ([]CrmBoostDoc, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	cur, err := s.coll.Find(ctx,
		bson.M{"status": CrmBoostPending},
		options.Find().SetSort(bson.D{{Key: "ts", Value: 1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []CrmBoostDoc
	return out, cur.All(ctx, &out)
}

// Resolve marks a boost row by junk_id with final status.
func (s *CrmBoostStore) Resolve(ctx context.Context, junkID, status, leadHashID, why string) error {
	if s == nil || junkID == "" || status == "" {
		return nil
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
	_, err := s.coll.UpdateOne(ctx, bson.M{"junk_id": junkID}, bson.M{"$set": set})
	return err
}

// ResolveByID marks a boost row by Mongo _id (crm-bot manual ops).
func (s *CrmBoostStore) ResolveByID(ctx context.Context, id primitive.ObjectID, status, leadHashID, why string) error {
	if s == nil || id.IsZero() || status == "" {
		return nil
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
	_, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set})
	return err
}
