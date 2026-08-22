package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const entityOutreachSystem = `You write short GTM outreach hooks for affiliate media buyers evaluating tracker products.
Use entity-level proof: multiple sightings, unified pain, buyer intent.
No PII, no fake familiarity. One sentence outreach_angle and one sentence entity_proof for CRM.`

var entityOutreachSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"outreach_angle": map[string]any{"type": "string"},
		"entity_proof":   map[string]any{"type": "string"},
	},
	"required": []any{"outreach_angle", "entity_proof"},
}

// EntityOutreachInput is context for cross-lead narrative generation.
type EntityOutreachInput struct {
	EntityID      string
	HeatTier      string
	SightingCount int
	SourceCount   int
	UnifiedPain   string
	BuyerIntent   string
	Sources       []string
}

// EntityOutreachResult is Gemini output for linked lead patches.
type EntityOutreachResult struct {
	OutreachAngle string
	EntityProof   string
}

type entityOutreachResponse struct {
	OutreachAngle string `json:"outreach_angle"`
	EntityProof   string `json:"entity_proof"`
}

// BuildEntityOutreach generates sales-ready angle and proof from entity graph context.
func (c *Client) BuildEntityOutreach(ctx context.Context, in EntityOutreachInput) (EntityOutreachResult, error) {
	payload, err := json.Marshal(map[string]any{
		"entity_id":       strings.TrimSpace(in.EntityID),
		"heat_tier":       strings.TrimSpace(in.HeatTier),
		"sighting_count":  in.SightingCount,
		"source_count":    in.SourceCount,
		"unified_pain":    strings.TrimSpace(in.UnifiedPain),
		"buyer_intent":    strings.TrimSpace(in.BuyerIntent),
		"source_families": in.Sources,
	})
	if err != nil {
		return EntityOutreachResult{}, err
	}
	parsed, err := classifyJSON[entityOutreachResponse](c, ctx, PriorityLow, entityOutreachSystem,
		fmt.Sprintf("Entity outreach narrative:\n\n%s", payload), entityOutreachSchema)
	if err != nil {
		return EntityOutreachResult{}, err
	}
	return EntityOutreachResult{
		OutreachAngle: strings.TrimSpace(parsed.OutreachAngle),
		EntityProof:   strings.TrimSpace(parsed.EntityProof),
	}, nil
}
