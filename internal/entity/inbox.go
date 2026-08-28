package entity

import "strings"

// InboxCard is the sales-facing view of one buyer graph node (not a single hash_id).
type InboxCard struct {
	EntityID           string   `json:"entity_id"`
	HeatTier           string   `json:"heat_tier"`
	HeatScore          int      `json:"heat_score"`
	SightingCount      int      `json:"sighting_count"`
	SourceCount        int      `json:"source_count"`
	SourceFamilies     []string `json:"source_families,omitempty"`
	UnifiedPain        string   `json:"unified_pain,omitempty"`
	BuyerIntent        string   `json:"buyer_intent,omitempty"`
	ActorConfidence    float64  `json:"actor_confidence,omitempty"`
	EntityProof        string   `json:"entity_proof,omitempty"`
	OutreachAngle      string   `json:"outreach_angle,omitempty"`
	BestContactChannel string   `json:"best_contact_channel,omitempty"`
	EngagePriority     int      `json:"engage_priority,omitempty"`
	NewLeadCount       int      `json:"new_lead_count"`
	PendingSuggestions int      `json:"pending_suggestions"`
	NeedsReview        bool     `json:"needs_review,omitempty"`
	ClassifyForce      bool     `json:"classify_force,omitempty"`
}

// BuildInboxCard maps an entity doc to a sales inbox row (outreach fields filled by store).
func BuildInboxCard(doc EntityDoc) InboxCard {
	card := InboxCard{
		EntityID:        doc.EntityID,
		HeatTier:        doc.HeatTier,
		HeatScore:       DisplayHeatScore(doc),
		SightingCount:   doc.SightingCount,
		SourceCount:     doc.SourceCount,
		SourceFamilies:  append([]string(nil), doc.SourceFamilies...),
		UnifiedPain:     strings.TrimSpace(doc.UnifiedPain),
		BuyerIntent:     strings.TrimSpace(doc.BuyerIntent),
		ActorConfidence: doc.ActorConfidence,
		NeedsReview:     doc.NeedsReview,
		ClassifyForce:   doc.ClassifyForce,
	}
	for _, s := range doc.ReviewSuggestions {
		if strings.EqualFold(strings.TrimSpace(s.Status), "pending") {
			card.PendingSuggestions++
		}
	}
	if card.EntityProof == "" {
		card.EntityProof = BuildEntityProofSummary(doc)
	}
	return card
}
