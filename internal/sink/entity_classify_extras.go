package sink

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/diag"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const classifyForcePollLimit = 32

// MarkClassifyForce flags an entity for warm-path Gemini re-classify (ops split, manual repair).
func (s *EntityStore) MarkClassifyForce(ctx context.Context, entityID string) error {
	if s == nil || entityID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := s.coll.UpdateOne(ctx,
		bson.M{"entity_id": entityID},
		bson.M{"$set": bson.M{"classify_force": true}},
	)
	return err
}

func (s *EntityStore) ListClassifyForce(ctx context.Context, limit int) ([]string, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = classifyForcePollLimit
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cur, err := s.coll.Find(ctx,
		bson.M{"classify_force": true},
		options.Find().SetProjection(bson.M{"entity_id": 1}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var ids []string
	for cur.Next(ctx) {
		var row struct {
			EntityID string `bson:"entity_id"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.EntityID != "" {
			ids = append(ids, row.EntityID)
		}
	}
	return ids, cur.Err()
}

const lowConfidenceHotPollLimit = 32

// ListLowConfidenceHotEntities returns hot/blazing entities with missing or low actor confidence.
func (s *EntityStore) ListLowConfidenceHotEntities(ctx context.Context, limit int) ([]string, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = lowConfidenceHotPollLimit
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"heat_tier": bson.M{"$in": []string{entity.HeatTierHot, entity.HeatTierBlazing}},
		"$or": bson.A{
			bson.M{"entity_classified_at": bson.M{"$exists": false}},
			bson.M{"actor_confidence": bson.M{"$lt": 0.5}},
		},
	}
	cur, err := s.coll.Find(ctx, filter,
		options.Find().SetProjection(bson.M{"entity_id": 1}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var ids []string
	for cur.Next(ctx) {
		var row struct {
			EntityID string `bson:"entity_id"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.EntityID != "" {
			ids = append(ids, row.EntityID)
		}
	}
	return ids, cur.Err()
}

func (s *EntityStore) ClearClassifyForce(ctx context.Context, entityID string) error {
	if s == nil || entityID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := s.coll.UpdateOne(ctx,
		bson.M{"entity_id": entityID},
		bson.M{"$unset": bson.M{"classify_force": ""}},
	)
	return err
}

// PatchLinkedLeadsHeat updates heat fields on all leads linked to an entity graph node.
func (s *MongoStore) PatchLinkedLeadsHeat(ctx context.Context, entityID string, hashIDs []string, sync entity.LinkedLeadHeatSync) error {
	if s == nil || entityID == "" {
		return nil
	}
	filter := linkedLeadsFilter(entityID, hashIDs)
	if filter == nil {
		return nil
	}
	set := bson.M{
		"entity_heat":           sync.HeatScore,
		"heat_tier":             sync.HeatTier,
		"entity_sighting_count": sync.SightingCount,
		"entity_source_count":   sync.SourceCount,
	}
	_, err := s.coll.UpdateMany(ctx, filter, bson.M{"$set": set})
	return err
}

// PatchLinkedLeadsOutreach sets entity-level GTM narrative on linked leads.
func (s *MongoStore) PatchLinkedLeadsOutreach(ctx context.Context, entityID string, hashIDs []string, sync entity.LinkedLeadOutreachSync) error {
	if s == nil || entityID == "" {
		return nil
	}
	filter := linkedLeadsFilter(entityID, hashIDs)
	if filter == nil {
		return nil
	}
	set := bson.M{}
	if v := strings.TrimSpace(sync.EntityProof); v != "" {
		set["entity_proof"] = v
	}
	if v := strings.TrimSpace(sync.OutreachAngle); v != "" {
		set["outreach_angle"] = v
	}
	if len(set) == 0 {
		return nil
	}
	_, err := s.coll.UpdateMany(ctx, filter, bson.M{"$set": set})
	return err
}

func linkedLeadsFilter(entityID string, hashIDs []string) bson.M {
	entityID = strings.TrimSpace(entityID)
	var clauses bson.A
	if entityID != "" {
		clauses = append(clauses, bson.M{"entity_id": entityID})
	}
	if len(hashIDs) > 0 {
		clauses = append(clauses, bson.M{"hash_id": bson.M{"$in": hashIDs}})
	}
	switch len(clauses) {
	case 0:
		return nil
	case 1:
		return clauses[0].(bson.M)
	default:
		return bson.M{"$or": clauses}
	}
}

// MaskedContactsSummary returns a masked contact line for entity classify prompts.
func (s *MongoStore) MaskedContactsSummary(ctx context.Context, hashID string) string {
	if s == nil || strings.TrimSpace(hashID) == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var doc LeadDoc
	err := s.coll.FindOne(ctx, bson.M{"hash_id": hashID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ""
		}
		return ""
	}
	if len(doc.Contacts) == 0 {
		return ""
	}
	contacts := make([]extract.Contact, 0, len(doc.Contacts))
	for _, c := range doc.Contacts {
		contacts = append(contacts, extract.Contact{Type: c.Type, Value: c.Value})
	}
	formatted := extract.FormatAll(contacts)
	if len(formatted) == 0 {
		return ""
	}
	masked := make([]string, 0, len(formatted))
	for _, line := range formatted {
		masked = append(masked, diag.MaskContact(line))
	}
	return strings.Join(masked, ", ")
}
