package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type EntitySplitResult struct {
	SourceEntityID string
	NewEntityID    string
	HashID         string
	SourceDeleted  bool
}

// SplitEntityHash detaches one lead hash from an entity graph node (ops only).
// Rolls back the new entity row if source update or lead patch fails.
func (s *LeadStore) SplitEntityHash(ctx context.Context, entityID, rawHash string) (EntitySplitResult, error) {
	if s == nil || s.entities == nil {
		return EntitySplitResult{}, fmt.Errorf("entity store not initialized")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return EntitySplitResult{}, fmt.Errorf("entity_id empty")
	}

	hashID, err := s.ResolveHashID(ctx, rawHash)
	if err != nil {
		return EntitySplitResult{}, err
	}

	source, err := s.GetEntity(ctx, entityID)
	if err != nil {
		return EntitySplitResult{}, err
	}
	if !entity.EntityHashKnown(&source, hashID) {
		return EntitySplitResult{}, entity.ErrSplitHashMissing
	}

	lead, err := s.getLeadForSplit(ctx, hashID)
	if err != nil {
		return EntitySplitResult{}, err
	}

	sight, ok := entitySightingByHash(source, hashID)
	if !ok {
		return EntitySplitResult{}, entity.ErrSplitHashMissing
	}

	sourceBackup := source
	splitIn := entity.SightingInputFromLead(leadSightingSourceFromDoc(lead), sight)
	heat := s.entityHeatConfig()
	splitResult, newDoc, err := entity.SplitHashFromEntity(&source, hashID, splitIn, heat)
	if err != nil {
		return EntitySplitResult{}, err
	}

	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	var existing entity.EntityDoc
	err = s.entities.FindOne(writeCtx, bson.M{"entity_id": newDoc.EntityID}).Decode(&existing)
	if err == nil && existing.EntityID != entityID {
		return EntitySplitResult{}, fmt.Errorf("target entity_id %s already exists; resolve merge manually", newDoc.EntityID)
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return EntitySplitResult{}, err
	}

	if _, err := s.entities.InsertOne(writeCtx, newDoc); err != nil {
		return EntitySplitResult{}, fmt.Errorf("insert split entity: %w", err)
	}

	if splitResult.SourceDeleted {
		if _, err := s.entities.DeleteOne(writeCtx, bson.M{"entity_id": entityID}); err != nil {
			s.rollbackSplitEntity(writeCtx, newDoc.EntityID, sourceBackup, splitResult.SourceDeleted)
			return EntitySplitResult{}, fmt.Errorf("delete empty source entity: %w", err)
		}
	} else {
		if _, err := s.entities.ReplaceOne(writeCtx, bson.M{"entity_id": entityID}, source); err != nil {
			s.rollbackSplitEntity(writeCtx, newDoc.EntityID, sourceBackup, splitResult.SourceDeleted)
			return EntitySplitResult{}, fmt.Errorf("update source entity: %w", err)
		}
	}

	leadPatch := bson.M{
		"entity_id":             newDoc.EntityID,
		"entity_heat":           newDoc.HeatScore,
		"heat_tier":             newDoc.HeatTier,
		"entity_sighting_count": newDoc.SightingCount,
		"entity_source_count":   newDoc.SourceCount,
	}
	if _, err := s.leads.UpdateOne(writeCtx, bson.M{"hash_id": hashID}, bson.M{"$set": leadPatch}); err != nil {
		s.rollbackSplitEntity(writeCtx, newDoc.EntityID, sourceBackup, splitResult.SourceDeleted)
		return EntitySplitResult{}, fmt.Errorf("update lead entity_id: %w", err)
	}

	if err := s.markClassifyForce(writeCtx, newDoc.EntityID); err != nil {
		return EntitySplitResult{}, fmt.Errorf("flag new entity classify: %w", err)
	}
	if !splitResult.SourceDeleted {
		if err := s.markClassifyForce(writeCtx, entityID); err != nil {
			return EntitySplitResult{}, fmt.Errorf("flag source entity classify: %w", err)
		}
	}

	return EntitySplitResult{
		SourceEntityID: splitResult.SourceEntityID,
		NewEntityID:    splitResult.NewEntityID,
		HashID:         splitResult.HashID,
		SourceDeleted:  splitResult.SourceDeleted,
	}, nil
}

func (s *LeadStore) rollbackSplitEntity(ctx context.Context, newEntityID string, sourceBackup entity.EntityDoc, sourceDeleted bool) {
	// Split is insert-then-update-lead; compensate when lead or source entity write fails.
	if s == nil || s.entities == nil {
		return
	}
	_, _ = s.entities.DeleteOne(ctx, bson.M{"entity_id": newEntityID})
	if sourceDeleted {
		_, _ = s.entities.InsertOne(ctx, sourceBackup)
		return
	}
	_, _ = s.entities.ReplaceOne(ctx, bson.M{"entity_id": sourceBackup.EntityID}, sourceBackup)
}

func (s *LeadStore) markClassifyForce(ctx context.Context, entityID string) error {
	if s == nil || s.entities == nil || entityID == "" {
		return nil
	}
	_, err := s.entities.UpdateOne(ctx,
		bson.M{"entity_id": entityID},
		bson.M{"$set": bson.M{"classify_force": true}},
	)
	return err
}

func (s *LeadStore) getLeadForSplit(ctx context.Context, hashID string) (sink.LeadDoc, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	var doc sink.LeadDoc
	err := s.leads.FindOne(queryCtx, bson.M{"hash_id": hashID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return sink.LeadDoc{}, ErrNotFound
	}
	if err != nil {
		return sink.LeadDoc{}, err
	}
	return doc, nil
}

func entitySightingByHash(doc entity.EntityDoc, hashID string) (entity.EntitySighting, bool) {
	for _, sight := range doc.Sightings {
		if sight.HashID == hashID {
			return sight, true
		}
	}
	return entity.EntitySighting{}, false
}

func leadSightingSourceFromDoc(lead sink.LeadDoc) entity.LeadSightingSource {
	return entity.LeadSightingSource{
		Source:       lead.Source,
		Title:        lead.Title,
		CompanyName:  lead.CompanyName,
		DisplayName:  lead.DisplayName,
		GravatarName: lead.GravatarName,
		Contacts:     storedContactsToExtract(lead.Contacts),
		Snippet:      lead.Snippet,
		Stack:        lead.Stack,
		Score:        lead.Score,
		Matched:      lead.Matched,
		PostedAt:     lead.PostedAt,
	}
}

func storedContactsToExtract(stored []sink.StoredContact) []extract.Contact {
	if len(stored) == 0 {
		return nil
	}
	out := make([]extract.Contact, 0, len(stored))
	for _, c := range stored {
		out = append(out, extract.Contact{Type: c.Type, Value: c.Value})
	}
	return out
}
