package entity

import "context"

// Recorder persists entity sightings (Mongo or in-memory).
type Recorder interface {
	RecordSighting(ctx context.Context, in SightingInput) (RecordResult, error)
}
