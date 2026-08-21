package entity

import (
	"strings"
	"time"

	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/validate"
)

func sightingTime(in SightingInput) time.Time {
	if !in.SeenAt.IsZero() {
		return in.SeenAt.UTC()
	}
	return time.Now().UTC()
}

func sightingSignals(stack []string, text string, matched []string) (hasPain bool, buyerRole bool) {
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

// NewDoc builds an entity document from the first sighting.
func NewDoc(entityID string, pk EntityKey, keys []EntityKey, in SightingInput) EntityDoc {
	now := sightingTime(in)
	hasPain, buyerRole := sightingSignals(in.Stack, in.Text, in.Matched)
	family := SourceFamily(in.Source)

	doc := EntityDoc{
		EntityID:       entityID,
		PrimaryKind:    pk.Kind,
		PrimaryValue:   pk.Value,
		AliasKeys:      AliasTokens(keys),
		HashIDs:        uniqStrings([]string{strings.TrimSpace(in.HashID)}),
		Sources:        uniqStrings([]string{strings.TrimSpace(in.Source)}),
		SourceFamilies: uniqStrings([]string{family}),
		Matched:        uniqStrings(in.Matched),
		Stack:          uniqStrings(in.Stack),
		FirstSeen:      now,
		LastSeen:       now,
		SightingCount:  1,
		SourceCount:    1,
		CanonicalHash:  strings.TrimSpace(in.HashID),
		BuyerRoleSeen:  buyerRole,
	}
	if hasPain {
		doc.PainHits = 1
	}
	return doc
}

// MergeSighting applies a sighting onto an existing entity document.
func MergeSighting(doc *EntityDoc, keys []EntityKey, in SightingInput) RecordResult {
	if doc == nil {
		return RecordResult{}
	}
	now := sightingTime(in)
	hasPain, buyerRole := sightingSignals(in.Stack, in.Text, in.Matched)
	family := SourceFamily(in.Source)

	newFamily := family != "" && !containsString(doc.SourceFamilies, family)
	doc.SightingCount++
	// Bump generation so Mongo replaceIfGeneration rejects concurrent writers on the same entity.
	doc.MergeGeneration++
	doc.LastSeen = now
	doc.AliasKeys = unionStrings(doc.AliasKeys, AliasTokens(keys))
	doc.HashIDs = unionStrings(doc.HashIDs, []string{strings.TrimSpace(in.HashID)})
	doc.Sources = unionStrings(doc.Sources, []string{strings.TrimSpace(in.Source)})
	if newFamily {
		doc.SourceFamilies = unionStrings(doc.SourceFamilies, []string{family})
	}
	doc.SourceCount = len(doc.SourceFamilies)
	doc.Matched = unionStrings(doc.Matched, in.Matched)
	doc.Stack = unionStrings(doc.Stack, in.Stack)
	if hasPain {
		doc.PainHits++
	}
	hadBuyerRole := doc.BuyerRoleSeen
	if buyerRole {
		doc.BuyerRoleSeen = true
	}
	if doc.CanonicalHash == "" {
		doc.CanonicalHash = strings.TrimSpace(in.HashID)
	} else if buyerRole && !hadBuyerRole {
		// Promote latest buyer-role sighting hash as canonical for cross-source hot patching.
		doc.CanonicalHash = strings.TrimSpace(in.HashID)
	}

	return RecordResult{
		EntityID:        doc.EntityID,
		SightingCount:   doc.SightingCount,
		SourceCount:     doc.SourceCount,
		NewSourceFamily: newFamily,
	}
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func uniqStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func unionStrings(base, add []string) []string {
	if len(add) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(add))
	out := make([]string, 0, len(base)+len(add))
	for _, v := range base {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range add {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
