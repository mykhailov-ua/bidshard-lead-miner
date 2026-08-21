package entity

import (
	"context"
	"errors"
	"strings"
	"time"
)

const DefaultMergeMaxAttempts = 4

const defaultMergeMaxAttempts = DefaultMergeMaxAttempts

// SightingBackend persists entity merge state for RecordSightingCore.
type SightingBackend interface {
	FindExisting(ctx context.Context, aliases []string, candidateID string) (*EntityDoc, error)
	InsertNew(ctx context.Context, doc EntityDoc) (duplicate bool, err error)
	ReplaceMerged(ctx context.Context, doc EntityDoc, generation int) (ok bool, err error)
}

func RecordSightingCore(
	ctx context.Context,
	in SightingInput,
	window time.Duration,
	maxAttempts int,
	backend SightingBackend,
) (RecordResult, error) {
	if backend == nil {
		return RecordResult{}, nil
	}
	keys := ResolveKeys(in.ResolveInput)
	if len(keys) == 0 {
		return RecordResult{}, nil
	}
	if strings.TrimSpace(in.HashID) == "" {
		return RecordResult{}, nil
	}
	if maxAttempts <= 0 {
		maxAttempts = defaultMergeMaxAttempts
	}

	aliases := AliasTokens(keys)
	candidateID := EntityID(keys)
	pk, _ := PrimaryKey(keys)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, done, err := recordSightingOnce(ctx, backend, keys, aliases, candidateID, pk, in, window)
		if err != nil || done {
			return result, err
		}
	}
	return RecordResult{}, errors.New("entity merge conflict after retries")
}

func recordSightingOnce(
	ctx context.Context,
	backend SightingBackend,
	keys []EntityKey,
	aliases []string,
	candidateID string,
	pk EntityKey,
	in SightingInput,
	window time.Duration,
) (RecordResult, bool, error) {
	if ab, ok := backend.(AtomicSightingBackend); ok {
		result, found, err := ab.MergeSightingAtomic(ctx, keys, aliases, candidateID, pk, in, window)
		if err != nil {
			return RecordResult{}, true, err
		}
		if found {
			return result, true, nil
		}
	}

	existing, err := backend.FindExisting(ctx, aliases, candidateID)
	if err != nil {
		return RecordResult{}, true, err
	}
	if existing != nil {
		gen := existing.MergeGeneration
		result := MergeSighting(existing, keys, in)
		ok, err := backend.ReplaceMerged(ctx, *existing, gen)
		if err != nil {
			return RecordResult{}, true, err
		}
		if !ok {
			return RecordResult{}, false, nil
		}
		return EnrichRecordResult(result, *existing, window), true, nil
	}

	doc := NewDoc(candidateID, pk, keys, in)
	dup, err := backend.InsertNew(ctx, doc)
	if err != nil {
		return RecordResult{}, true, err
	}
	if dup {
		return RecordResult{}, false, nil
	}
	return EnrichRecordResult(RecordResult{
		EntityID:      candidateID,
		SightingCount: 1,
		SourceCount:   1,
		NewEntity:     true,
	}, doc, window), true, nil
}
