package entity

import "time"

// EntityDoc is the Mongo document for cross-source entity aggregation.
type EntityDoc struct {
	EntityID           string           `bson:"entity_id" json:"entity_id"`
	PrimaryKind        string           `bson:"primary_kind" json:"primary_kind"`
	PrimaryValue       string           `bson:"primary_value" json:"primary_value"`
	AliasKeys          []string         `bson:"alias_keys" json:"alias_keys"`
	HashIDs            []string         `bson:"hash_ids" json:"hash_ids"`
	Sources            []string         `bson:"sources" json:"sources"`
	SourceFamilies     []string         `bson:"source_families" json:"source_families"`
	Matched            []string         `bson:"matched" json:"matched"`
	Stack              []string         `bson:"stack" json:"stack"`
	FirstSeen          time.Time        `bson:"first_seen" json:"first_seen"`
	LastSeen           time.Time        `bson:"last_seen" json:"last_seen"`
	SightingCount      int              `bson:"sighting_count" json:"sighting_count"`
	SourceCount        int              `bson:"source_count" json:"source_count"`
	PainHits           int              `bson:"pain_hits" json:"pain_hits"`
	BuyerRoleSeen      bool             `bson:"buyer_role_seen" json:"buyer_role_seen"`
	CanonicalHash      string           `bson:"canonical_hash_id" json:"canonical_hash_id"`
	HeatScore          float64          `bson:"heat_score,omitempty" json:"heat_score,omitempty"`
	HeatTier           string           `bson:"heat_tier,omitempty" json:"heat_tier,omitempty"`
	LastHeatAt         time.Time        `bson:"last_heat_at,omitempty" json:"last_heat_at,omitempty"`
	FreshSightingAt    time.Time        `bson:"fresh_sighting_at,omitempty" json:"fresh_sighting_at,omitempty"`
	UnifiedPain        string           `bson:"unified_pain,omitempty" json:"unified_pain,omitempty"`
	ActorConfidence    float64          `bson:"actor_confidence,omitempty" json:"actor_confidence,omitempty"`
	BuyerIntent        string           `bson:"buyer_intent,omitempty" json:"buyer_intent,omitempty"`
	NeedsReview        bool             `bson:"needs_review,omitempty" json:"needs_review,omitempty"`
	EntityClassifiedAt time.Time        `bson:"entity_classified_at,omitempty" json:"entity_classified_at,omitempty"`
	MergeGeneration    int              `bson:"merge_generation" json:"merge_generation"`
	Sightings          []EntitySighting `bson:"sightings,omitempty" json:"sightings,omitempty"`
}

// SightingInput is one pipeline observation linked to an entity.
type SightingInput struct {
	ResolveInput
	HashID   string
	Matched  []string
	Stack    []string
	Text     string
	Score    int
	PostedAt time.Time
	SeenAt   time.Time
}

// RecordResult summarizes a recorded sighting.
type RecordResult struct {
	EntityID        string
	SightingCount   int
	SourceCount     int
	NewSourceFamily bool
	NewEntity       bool
	CanonicalHash   string
	CrossSourceHot  bool
	HeatScore       float64
	HeatTier        string
}
