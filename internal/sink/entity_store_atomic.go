package sink

import (
	"context"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/entity"
)

// MergeSightingAtomic merges a new sighting and persists with merge_generation CAS.
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

	doc := *existing
	gen := doc.MergeGeneration
	mergeResult := entity.MergeSightingWithHeat(&doc, keys, in, heat, time.Now().UTC())

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
		EntityID:        doc.EntityID,
		SightingCount:   doc.SightingCount,
		SourceCount:     doc.SourceCount,
		NewSourceFamily: mergeResult.NewSourceFamily,
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
