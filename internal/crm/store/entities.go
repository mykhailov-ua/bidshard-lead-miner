package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultEntityListLimit = 20
	maxEntityListLimit     = 50
)

type EntityListFilter struct {
	MinTier string
	Limit   int64
}

type EntityListResult struct {
	Entities []entity.EntityDoc `json:"entities"`
}

func (s *LeadStore) ListEntities(ctx context.Context, filter EntityListFilter) (EntityListResult, error) {
	if s == nil || s.entities == nil {
		return EntityListResult{}, fmt.Errorf("entity store not initialized")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultEntityListLimit
	}
	if limit > maxEntityListLimit {
		limit = maxEntityListLimit
	}

	q := bson.M{}
	if tiers := entity.HeatTiersAtOrAbove(filter.MinTier); len(tiers) > 0 {
		q["heat_tier"] = bson.M{"$in": tiers}
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	opts := options.Find().
		SetSort(bson.D{
			{Key: "heat_score", Value: -1},
			{Key: "last_seen", Value: -1},
		}).
		SetLimit(limit)

	cur, err := s.entities.Find(queryCtx, q, opts)
	if err != nil {
		return EntityListResult{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	docs := make([]entity.EntityDoc, 0, limit)
	for cur.Next(queryCtx) {
		var doc entity.EntityDoc
		if err := cur.Decode(&doc); err != nil {
			return EntityListResult{}, err
		}
		docs = append(docs, doc)
	}
	if err := cur.Err(); err != nil {
		return EntityListResult{}, err
	}
	return EntityListResult{Entities: docs}, nil
}

func (s *LeadStore) GetEntity(ctx context.Context, entityID string) (entity.EntityDoc, error) {
	if s == nil || s.entities == nil {
		return entity.EntityDoc{}, fmt.Errorf("entity store not initialized")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return entity.EntityDoc{}, fmt.Errorf("entity_id empty")
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	var doc entity.EntityDoc
	err := s.entities.FindOne(queryCtx, bson.M{"entity_id": entityID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entity.EntityDoc{}, ErrNotFound
	}
	if err != nil {
		return entity.EntityDoc{}, err
	}
	return doc, nil
}

func (s *LeadStore) ListEntityLeads(ctx context.Context, entityID string, limit int64) ([]sink.LeadDoc, error) {
	if s == nil || s.leads == nil {
		return nil, fmt.Errorf("lead store not initialized")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return nil, fmt.Errorf("entity_id empty")
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	doc, err := s.GetEntity(ctx, entityID)
	if err != nil {
		return nil, err
	}

	query := bson.M{"entity_id": entityID}
	if len(doc.HashIDs) > 0 {
		query = bson.M{
			"$or": bson.A{
				bson.M{"entity_id": entityID},
				bson.M{"hash_id": bson.M{"$in": doc.HashIDs}},
			},
		}
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	opts := options.Find().
		SetSort(bson.D{
			{Key: "entity_heat", Value: -1},
			{Key: "score", Value: -1},
			{Key: "hash_id", Value: 1},
		}).
		SetLimit(limit).
		SetProjection(leadCardProjection())

	cur, err := s.leads.Find(queryCtx, query, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var docs []sink.LeadDoc
	for cur.Next(queryCtx) {
		var lead sink.LeadDoc
		if err := cur.Decode(&lead); err != nil {
			return nil, err
		}
		docs = append(docs, lead)
	}
	return docs, cur.Err()
}

// EntitiesEnabled reports whether the CRM store can query the entities collection.
func (s *LeadStore) EntitiesEnabled() bool {
	return s != nil && s.entities != nil
}
