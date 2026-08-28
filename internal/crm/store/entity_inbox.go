package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultEntityInboxLimit = 20
	maxEntityInboxLimit     = 50
)

// EntityInboxFilter selects heat-ranked buyers for the sales inbox.
type EntityInboxFilter struct {
	MinTier           string
	MinSightings      int
	OnlyNeedsWork     bool // needs_review, classify_force, or pending link suggestions
	MinEngagePriority int
	Limit             int64
}

// EntityInboxResult is a page of entity-centric inbox cards.
type EntityInboxResult struct {
	Entities []entity.InboxCard `json:"entities"`
}

func (s *LeadStore) ListEntityInbox(ctx context.Context, filter EntityInboxFilter) (EntityInboxResult, error) {
	if s == nil || s.entities == nil {
		return EntityInboxResult{}, fmt.Errorf("entity store not initialized")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultEntityInboxLimit
	}
	if limit > maxEntityInboxLimit {
		limit = maxEntityInboxLimit
	}
	minSightings := filter.MinSightings
	if minSightings <= 0 {
		minSightings = 2
	}

	q := bson.M{"sighting_count": bson.M{"$gte": minSightings}}
	if tiers := entity.HeatTiersAtOrAbove(filter.MinTier); len(tiers) > 0 {
		q["heat_tier"] = bson.M{"$in": tiers}
	}
	if filter.OnlyNeedsWork {
		q["$or"] = bson.A{
			bson.M{"needs_review": true},
			bson.M{"classify_force": true},
			bson.M{"review_suggestions": bson.M{"$elemMatch": bson.M{"status": "pending"}}},
		}
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	fetchLimit := limit
	if filter.MinEngagePriority > 0 && fetchLimit < maxEntityInboxLimit {
		fetchLimit = limit * 3
		if fetchLimit > maxEntityInboxLimit*2 {
			fetchLimit = maxEntityInboxLimit * 2
		}
	}

	cur, err := s.entities.Find(queryCtx, q,
		options.Find().
			SetSort(bson.D{
				{Key: "heat_score", Value: -1},
				{Key: "sighting_count", Value: -1},
				{Key: "last_seen", Value: -1},
			}).
			SetLimit(fetchLimit),
	)
	if err != nil {
		return EntityInboxResult{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var cards []entity.InboxCard
	for cur.Next(queryCtx) {
		var doc entity.EntityDoc
		if err := cur.Decode(&doc); err != nil {
			return EntityInboxResult{}, err
		}
		card := entity.BuildInboxCard(doc)
		s.enrichInboxFromLeads(queryCtx, &card, doc, filter.MinEngagePriority)
		if filter.MinEngagePriority > 0 && card.EngagePriority < filter.MinEngagePriority {
			continue
		}
		cards = append(cards, card)
		if int64(len(cards)) >= limit {
			break
		}
	}
	if err := cur.Err(); err != nil {
		return EntityInboxResult{}, err
	}
	return EntityInboxResult{Entities: cards}, nil
}

func (s *LeadStore) GetEntityInbox(ctx context.Context, entityID string) (entity.InboxCard, error) {
	doc, err := s.GetEntity(ctx, entityID)
	if err != nil {
		return entity.InboxCard{}, err
	}
	card := entity.BuildInboxCard(doc)
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	s.enrichInboxFromLeads(queryCtx, &card, doc, 0)
	return card, nil
}

func (s *LeadStore) enrichInboxFromLeads(ctx context.Context, card *entity.InboxCard, doc entity.EntityDoc, minEngagePriority int) {
	if s == nil || s.leads == nil || card == nil {
		return
	}
	filter := linkedLeadsFilter(doc.EntityID, doc.HashIDs)
	if filter == nil {
		return
	}
	filter["status"] = "new"
	if minEngagePriority > 0 {
		filter["engage_priority"] = bson.M{"$gte": minEngagePriority}
	}

	count, err := s.leads.CountDocuments(ctx, filter)
	if err == nil {
		card.NewLeadCount = int(count)
	}

	opts := options.FindOne().
		SetSort(bson.D{
			{Key: "engage_priority", Value: -1},
			{Key: "entity_heat", Value: -1},
			{Key: "score", Value: -1},
		}).
		SetProjection(bson.M{
			"entity_proof":    1,
			"outreach_angle":  1,
			"contact_channel": 1,
			"engage_priority": 1,
		})

	var best struct {
		EntityProof    string `bson:"entity_proof"`
		OutreachAngle  string `bson:"outreach_angle"`
		ContactChannel string `bson:"contact_channel"`
		EngagePriority int    `bson:"engage_priority"`
	}
	if err := s.leads.FindOne(ctx, filter, opts).Decode(&best); err != nil {
		return
	}
	if card.EntityProof == "" {
		card.EntityProof = strings.TrimSpace(best.EntityProof)
	}
	if card.OutreachAngle == "" {
		card.OutreachAngle = strings.TrimSpace(best.OutreachAngle)
	}
	if card.BestContactChannel == "" {
		card.BestContactChannel = strings.TrimSpace(best.ContactChannel)
	}
	if best.EngagePriority > card.EngagePriority {
		card.EngagePriority = best.EngagePriority
	}
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

// ListPendingReviewSuggestions returns entities with pending merge/split suggestions.
func (s *LeadStore) ListPendingReviewSuggestions(ctx context.Context, limit int64) ([]entity.EntityDoc, error) {
	if s == nil || s.entities == nil {
		return nil, fmt.Errorf("entity store not initialized")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > maxEntityInboxLimit {
		limit = maxEntityInboxLimit
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	cur, err := s.entities.Find(queryCtx,
		bson.M{"review_suggestions": bson.M{"$elemMatch": bson.M{"status": "pending"}}},
		options.Find().
			SetSort(bson.D{{Key: "last_seen", Value: -1}}).
			SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var docs []entity.EntityDoc
	if err := cur.All(queryCtx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// MarkEntityClassifyForce queues warm-path Gemini re-classify (ops split/repair).
func (s *LeadStore) MarkEntityClassifyForce(ctx context.Context, entityID string) error {
	return s.markClassifyForce(ctx, strings.TrimSpace(entityID))
}
