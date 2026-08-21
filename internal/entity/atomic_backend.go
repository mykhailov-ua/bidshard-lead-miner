package entity

import (
	"context"
	"time"
)

// AtomicSightingBackend merges sightings in one store round-trip (Mongo pipeline update).
type AtomicSightingBackend interface {
	SightingBackend
	MergeSightingAtomic(
		ctx context.Context,
		keys []EntityKey,
		aliases []string,
		candidateID string,
		pk EntityKey,
		in SightingInput,
		window time.Duration,
	) (RecordResult, bool, error)
}
