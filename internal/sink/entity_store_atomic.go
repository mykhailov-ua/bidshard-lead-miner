package sink

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/validate"
)

// MergeSightingAtomic applies a sighting with a single Mongo pipeline update (no RMW replace).
func (s *EntityStore) MergeSightingAtomic(
	ctx context.Context,
	keys []entity.EntityKey,
	aliases []string,
	candidateID string,
	pk entity.EntityKey,
	in entity.SightingInput,
	window time.Duration,
	heat entity.HeatConfig,
) (entity.RecordResult, bool, error) {
	if s == nil {
		return entity.RecordResult{}, false, nil
	}

	existing, err := s.FindExisting(ctx, aliases, candidateID)
	if err != nil {
		return entity.RecordResult{}, false, err
	}
	if existing == nil {
		return entity.RecordResult{}, false, nil
	}

	hashID := strings.TrimSpace(in.HashID)
	if entity.EntityHashKnown(existing, hashID) {
		return s.mergeEntitySightingRefresh(ctx, existing, in, window, heat)
	}

	now := entitySightingEventTime(in)
	hasPain, buyerRole := atomicSightingSignals(in.Stack, in.Text, in.Matched)
	family := entity.SourceFamily(in.Source)
	newFamily := family != "" && !containsString(existing.SourceFamilies, family)
	gen := existing.MergeGeneration

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			"sighting_count":   bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$sighting_count", 0}}, 1}},
			"merge_generation": bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$merge_generation", 0}}, 1}},
			"last_seen":        now,
			"alias_keys":       bson.M{"$setUnion": bson.A{"$alias_keys", aliases}},
			"hash_ids":         bson.M{"$setUnion": bson.A{"$hash_ids", bson.A{hashID}}},
			"sources":          bson.M{"$setUnion": bson.A{"$sources", bson.A{strings.TrimSpace(in.Source)}}},
			"source_families":  bson.M{"$setUnion": bson.A{"$source_families", bson.A{family}}},
			"matched":          bson.M{"$setUnion": bson.A{"$matched", in.Matched}},
			"stack":            bson.M{"$setUnion": bson.A{"$stack", in.Stack}},
			"buyer_role_seen":  bson.M{"$or": bson.A{"$buyer_role_seen", buyerRole}},
			"canonical_hash_id": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$canonical_hash_id", ""}}, ""}},
					hashID,
					bson.M{
						"$cond": bson.A{
							bson.M{"$and": bson.A{buyerRole, bson.M{"$eq": bson.A{"$buyer_role_seen", false}}}},
							hashID,
							"$canonical_hash_id",
						},
					},
				},
			},
		}}},
		bson.D{{Key: "$set", Value: bson.M{
			"source_count": bson.M{"$size": "$source_families"},
		}}},
	}

	if hasPain {
		pipeline = append(pipeline, bson.D{{Key: "$set", Value: bson.M{
			"pain_hits": bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$pain_hits", 0}}, 1}},
		}}})
	}

	var doc entity.EntityDoc
	err = s.coll.FindOneAndUpdate(
		ctx,
		bson.M{"entity_id": existing.EntityID},
		pipeline,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return entity.RecordResult{}, false, nil
		}
		return entity.RecordResult{}, false, err
	}

	entity.PatchSightingTimeline(&doc, in)
	entity.RecomputeEntityHeat(&doc, heat, time.Now().UTC())

	ok, err := s.ReplaceMerged(ctx, doc, gen)
	if err != nil {
		return entity.RecordResult{}, false, err
	}
	if !ok {
		return entity.RecordResult{}, false, nil
	}

	result := entity.RecordResult{
		EntityID:        doc.EntityID,
		SightingCount:   doc.SightingCount,
		SourceCount:     doc.SourceCount,
		NewSourceFamily: newFamily,
	}
	return entity.EnrichRecordResult(result, doc, window, heat), true, nil
}

func (s *EntityStore) mergeEntitySightingRefresh(
	ctx context.Context,
	existing *entity.EntityDoc,
	in entity.SightingInput,
	window time.Duration,
	heat entity.HeatConfig,
) (entity.RecordResult, bool, error) {
	if existing == nil {
		return entity.RecordResult{}, false, nil
	}
	doc := *existing
	gen := doc.MergeGeneration
	entity.RefreshEntitySighting(&doc, in)
	entity.RecomputeEntityHeat(&doc, heat, time.Now().UTC())

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ok, err := s.ReplaceMerged(ctx, doc, gen)
	if err != nil {
		return entity.RecordResult{}, false, err
	}
	if !ok {
		return entity.RecordResult{}, false, nil
	}
	result := entity.RecordResult{
		EntityID:      doc.EntityID,
		SightingCount: doc.SightingCount,
		SourceCount:   doc.SourceCount,
	}
	return entity.EnrichRecordResult(result, doc, window, heat), true, nil
}

func entitySightingEventTime(in entity.SightingInput) time.Time {
	if !in.PostedAt.IsZero() {
		return in.PostedAt.UTC()
	}
	if !in.SeenAt.IsZero() {
		return in.SeenAt.UTC()
	}
	return time.Now().UTC()
}

func entityHashKnown(doc *entity.EntityDoc, hashID string) bool {
	return entity.EntityHashKnown(doc, hashID)
}

func atomicSightingSignals(stack []string, text string, matched []string) (hasPain bool, buyerRole bool) {
	hasPain = len(matched) > 0 || validate.HasPainContext(text)
	_, tags := scoring.PilotQualified("", stack, text)
	for _, tag := range tags {
		if tag == "pilot-buyer-role" {
			buyerRole = true
			break
		}
	}
	return hasPain, buyerRole
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
