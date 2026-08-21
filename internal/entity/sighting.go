package entity

import (
	"sort"
	"strings"
	"time"
)

const (
	// MaxEntitySightings caps stored timeline rows per entity (newest by event time).
	MaxEntitySightings = 20
	maxSightingSnippet = 500
)

// EntitySighting is one linked observation on an entity timeline.
type EntitySighting struct {
	HashID   string    `bson:"hash_id" json:"hash_id"`
	Source   string    `bson:"source" json:"source"`
	Family   string    `bson:"family" json:"family"`
	PostedAt time.Time `bson:"posted_at,omitempty" json:"posted_at,omitempty"`
	SeenAt   time.Time `bson:"seen_at" json:"seen_at"`
	Matched  []string  `bson:"matched,omitempty" json:"matched,omitempty"`
	Snippet  string    `bson:"snippet,omitempty" json:"snippet,omitempty"`
	Score    int       `bson:"score,omitempty" json:"score,omitempty"`
}

// sightingEventTime is the timeline instant used for first_seen/last_seen and ordering.
// Prefer forum post time over crawl time.
func sightingEventTime(in SightingInput) time.Time {
	if !in.PostedAt.IsZero() {
		return in.PostedAt.UTC()
	}
	if !in.SeenAt.IsZero() {
		return in.SeenAt.UTC()
	}
	return time.Now().UTC()
}

func sightingSeenAt(in SightingInput) time.Time {
	if !in.SeenAt.IsZero() {
		return in.SeenAt.UTC()
	}
	return time.Now().UTC()
}

func buildEntitySighting(in SightingInput) EntitySighting {
	return EntitySighting{
		HashID:   strings.TrimSpace(in.HashID),
		Source:   strings.TrimSpace(in.Source),
		Family:   SourceFamily(in.Source),
		PostedAt: in.PostedAt.UTC(),
		SeenAt:   sightingSeenAt(in),
		Matched:  uniqStrings(in.Matched),
		Snippet:  truncateSightingSnippet(in.Text),
		Score:    in.Score,
	}
}

func truncateSightingSnippet(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if len(text) <= maxSightingSnippet {
		return text
	}
	return text[:maxSightingSnippet]
}

func updateEntitySeenBounds(doc *EntityDoc, in SightingInput) {
	if doc == nil {
		return
	}
	event := sightingEventTime(in)
	if doc.FirstSeen.IsZero() || event.Before(doc.FirstSeen) {
		doc.FirstSeen = event
	}
	if event.After(doc.LastSeen) {
		doc.LastSeen = event
	}
}

// appendOrRefreshEntitySighting adds a timeline row or refreshes SeenAt for a known hash_id.
// Returns true when a new sighting row was appended.
func appendOrRefreshEntitySighting(doc *EntityDoc, in SightingInput) bool {
	if doc == nil {
		return false
	}
	hashID := strings.TrimSpace(in.HashID)
	if hashID == "" {
		return false
	}
	seenAt := sightingSeenAt(in)
	for i := range doc.Sightings {
		if doc.Sightings[i].HashID != hashID {
			continue
		}
		doc.Sightings[i].SeenAt = seenAt
		if !in.PostedAt.IsZero() {
			doc.Sightings[i].PostedAt = in.PostedAt.UTC()
		}
		if in.Score > 0 {
			doc.Sightings[i].Score = in.Score
		}
		if len(in.Matched) > 0 {
			doc.Sightings[i].Matched = uniqStrings(in.Matched)
		}
		if strings.TrimSpace(in.Text) != "" {
			doc.Sightings[i].Snippet = truncateSightingSnippet(in.Text)
		}
		trimEntitySightings(doc)
		return false
	}
	doc.Sightings = append(doc.Sightings, buildEntitySighting(in))
	trimEntitySightings(doc)
	return true
}

func sightingEventTimeOf(s EntitySighting) time.Time {
	if !s.PostedAt.IsZero() {
		return s.PostedAt.UTC()
	}
	if !s.SeenAt.IsZero() {
		return s.SeenAt.UTC()
	}
	return time.Time{}
}

func trimEntitySightings(doc *EntityDoc) {
	if doc == nil || len(doc.Sightings) <= MaxEntitySightings {
		return
	}
	sort.Slice(doc.Sightings, func(i, j int) bool {
		ti := sightingEventTimeOf(doc.Sightings[i])
		tj := sightingEventTimeOf(doc.Sightings[j])
		if ti.Equal(tj) {
			return doc.Sightings[i].HashID > doc.Sightings[j].HashID
		}
		return ti.After(tj)
	})
	doc.Sightings = doc.Sightings[:MaxEntitySightings]
}

func entityHashKnown(doc *EntityDoc, hashID string) bool {
	if doc == nil {
		return false
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return false
	}
	return containsString(doc.HashIDs, hashID)
}

// EntityHashKnown reports whether hash_id was already linked to the entity.
func EntityHashKnown(doc *EntityDoc, hashID string) bool {
	return entityHashKnown(doc, hashID)
}

// PatchSightingTimeline appends or refreshes sightings[] and updates first/last seen bounds.
func PatchSightingTimeline(doc *EntityDoc, in SightingInput) {
	appendOrRefreshEntitySighting(doc, in)
	updateEntitySeenBounds(doc, in)
}

// RefreshEntitySighting updates timeline metadata for a duplicate hash_id sighting.
func RefreshEntitySighting(doc *EntityDoc, in SightingInput) RecordResult {
	return refreshEntitySighting(doc, in)
}
