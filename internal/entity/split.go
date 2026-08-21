package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/extract"
)

var (
	ErrSplitHashMissing   = errors.New("hash_id not linked to entity")
	ErrSplitKeysEmpty     = errors.New("no entity keys resolved for split lead")
	ErrSplitEntityMissing = errors.New("entity document nil")
)

// LeadSightingSource supplies persisted lead fields to rebuild a sighting after ops split.
type LeadSightingSource struct {
	Source       string
	Title        string
	CompanyName  string
	DisplayName  string
	GravatarName string
	Contacts     []extract.Contact
	Snippet      string
	Stack        []string
	Score        int
	Matched      []string
	PostedAt     time.Time
}

// SightingInputFromLead rebuilds sighting input from a stored lead and entity timeline row.
// Forum username is not persisted on LeadDoc; EnrichForumIdentity falls back to title.
func SightingInputFromLead(lead LeadSightingSource, sight EntitySighting) SightingInput {
	resolve := ResolveInput{
		CompanyName:  lead.CompanyName,
		DisplayName:  lead.DisplayName,
		GravatarName: lead.GravatarName,
		Source:       lead.Source,
		Contacts:     lead.Contacts,
	}
	resolve = EnrichForumIdentity(resolve, "", lead.Title, "")

	text := strings.TrimSpace(lead.Snippet)
	if text == "" {
		text = strings.TrimSpace(sight.Snippet)
	}
	matched := lead.Matched
	if len(matched) == 0 {
		matched = sight.Matched
	}
	score := lead.Score
	if score <= 0 {
		score = sight.Score
	}
	postedAt := lead.PostedAt
	if postedAt.IsZero() {
		postedAt = sight.PostedAt
	}
	seenAt := sight.SeenAt
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}

	return SightingInput{
		ResolveInput: resolve,
		HashID:       sight.HashID,
		Matched:      matched,
		Stack:        lead.Stack,
		Text:         text,
		Score:        score,
		PostedAt:     postedAt,
		SeenAt:       seenAt,
	}
}

// SplitHashResult is the outcome of detaching one hash_id into a new entity graph node.
type SplitHashResult struct {
	SourceEntityID string
	NewEntityID    string
	HashID         string
	SourceDeleted  bool
}

// SplitHashFromEntity removes hashID from doc and builds a standalone entity for that sighting.
// Source entity fields are rebuilt from remaining sightings; Gemini fields are cleared on source.
func SplitHashFromEntity(doc *EntityDoc, hashID string, in SightingInput, heat HeatConfig) (SplitHashResult, EntityDoc, error) {
	if doc == nil {
		return SplitHashResult{}, EntityDoc{}, ErrSplitEntityMissing
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return SplitHashResult{}, EntityDoc{}, fmt.Errorf("hash_id empty")
	}
	if !entityHashKnown(doc, hashID) {
		return SplitHashResult{}, EntityDoc{}, ErrSplitHashMissing
	}

	keys := ResolveKeys(in.ResolveInput)
	if len(keys) == 0 {
		return SplitHashResult{}, EntityDoc{}, ErrSplitKeysEmpty
	}
	newEntityID := EntityIDForSplit(keys, hashID, doc.EntityID)
	if newEntityID == "" {
		return SplitHashResult{}, EntityDoc{}, ErrSplitKeysEmpty
	}
	pk, ok := PrimaryKey(keys)
	if !ok {
		return SplitHashResult{}, EntityDoc{}, ErrSplitKeysEmpty
	}

	now := time.Now().UTC()
	sourceID := doc.EntityID

	removeHashFromEntity(doc, hashID)
	doc.MergeGeneration++
	// Stale Gemini classification would mis-rank the shrunk graph; force re-classify.
	clearEntityClassification(doc)
	// Decrement counters are unsafe after merge growth; rebuild from sightings[].
	rebuildEntityAggregates(doc, heat, now)

	newDoc := NewDocWithHeat(newEntityID, pk, keys, in, heat, now)
	newDoc.NeedsReview = false

	result := SplitHashResult{
		SourceEntityID: sourceID,
		NewEntityID:    newEntityID,
		HashID:         hashID,
		SourceDeleted:  len(doc.HashIDs) == 0,
	}
	return result, newDoc, nil
}

func removeHashFromEntity(doc *EntityDoc, hashID string) {
	doc.HashIDs = removeString(doc.HashIDs, hashID)
	remaining := make([]EntitySighting, 0, len(doc.Sightings))
	for _, s := range doc.Sightings {
		if s.HashID == hashID {
			continue
		}
		remaining = append(remaining, s)
	}
	doc.Sightings = remaining
	if doc.CanonicalHash == hashID {
		doc.CanonicalHash = ""
	}
}

func clearEntityClassification(doc *EntityDoc) {
	doc.UnifiedPain = ""
	doc.ActorConfidence = 0
	doc.BuyerIntent = ""
	doc.NeedsReview = false
	doc.EntityClassifiedAt = time.Time{}
}

func rebuildEntityAggregates(doc *EntityDoc, heat HeatConfig, now time.Time) {
	if doc == nil {
		return
	}
	// alias_keys are intentionally not pruned: we cannot tell which aliases belonged
	// only to the detached hash without re-resolving every remaining lead offline.
	doc.Sources = nil
	doc.SourceFamilies = nil
	doc.Matched = nil
	doc.PainHits = 0
	doc.BuyerRoleSeen = false
	doc.FirstSeen = time.Time{}
	doc.LastSeen = time.Time{}

	hashSet := make(map[string]struct{}, len(doc.Sightings))
	for _, sight := range doc.Sightings {
		hashID := strings.TrimSpace(sight.HashID)
		if hashID != "" {
			hashSet[hashID] = struct{}{}
		}
		if src := strings.TrimSpace(sight.Source); src != "" {
			doc.Sources = unionStrings(doc.Sources, []string{src})
		}
		if family := strings.TrimSpace(sight.Family); family != "" {
			doc.SourceFamilies = unionStrings(doc.SourceFamilies, []string{family})
		} else if family := SourceFamily(sight.Source); family != "" {
			doc.SourceFamilies = unionStrings(doc.SourceFamilies, []string{family})
		}
		doc.Matched = unionStrings(doc.Matched, sight.Matched)
		hasPain, buyerRole := sightingSignals(nil, sight.Snippet, sight.Matched)
		if hasPain {
			doc.PainHits++
		}
		if buyerRole {
			doc.BuyerRoleSeen = true
		}
		event := sightingEventTimeOf(sight)
		if !event.IsZero() {
			if doc.FirstSeen.IsZero() || event.Before(doc.FirstSeen) {
				doc.FirstSeen = event
			}
			if event.After(doc.LastSeen) {
				doc.LastSeen = event
			}
		}
	}
	doc.HashIDs = make([]string, 0, len(hashSet))
	for hashID := range hashSet {
		doc.HashIDs = append(doc.HashIDs, hashID)
	}
	doc.SightingCount = len(doc.HashIDs)
	doc.SourceCount = len(doc.SourceFamilies)
	if doc.CanonicalHash == "" && len(doc.HashIDs) > 0 {
		doc.CanonicalHash = doc.HashIDs[0]
	}
	RecomputeEntityHeat(doc, heat, now)
}

func removeString(list []string, target string) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		if v == target {
			continue
		}
		out = append(out, v)
	}
	return out
}
