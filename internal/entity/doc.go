package entity

import "time"

// EntityDoc is the Mongo document for cross-source entity aggregation.
type EntityDoc struct {
	EntityID        string    `bson:"entity_id" json:"entity_id"`
	PrimaryKind     string    `bson:"primary_kind" json:"primary_kind"`
	PrimaryValue    string    `bson:"primary_value" json:"primary_value"`
	AliasKeys       []string  `bson:"alias_keys" json:"alias_keys"`
	HashIDs         []string  `bson:"hash_ids" json:"hash_ids"`
	Sources         []string  `bson:"sources" json:"sources"`
	SourceFamilies  []string  `bson:"source_families" json:"source_families"`
	Matched         []string  `bson:"matched" json:"matched"`
	Stack           []string  `bson:"stack" json:"stack"`
	FirstSeen       time.Time `bson:"first_seen" json:"first_seen"`
	LastSeen        time.Time `bson:"last_seen" json:"last_seen"`
	SightingCount   int       `bson:"sighting_count" json:"sighting_count"`
	SourceCount     int       `bson:"source_count" json:"source_count"`
	PainHits        int       `bson:"pain_hits" json:"pain_hits"`
	BuyerRoleSeen   bool      `bson:"buyer_role_seen" json:"buyer_role_seen"`
	CanonicalHash   string    `bson:"canonical_hash_id" json:"canonical_hash_id"`
	MergeGeneration int       `bson:"merge_generation" json:"merge_generation"`
}

// SightingInput is one pipeline observation linked to an entity.
type SightingInput struct {
	ResolveInput
	HashID  string
	Matched []string
	Stack   []string
	Text    string
	SeenAt  time.Time
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
}
