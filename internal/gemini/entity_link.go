package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const entityLinkSystem = `You review affiliate buyer entity graph pairs that share a domain alias.
Decide whether ops should merge, split, or keep separate.
merge: same buyer, false split; split: different buyers incorrectly merged; keep: shared domain is expected (agency, network).
Output JSON only.`

var entityLinkSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"action": map[string]any{
			"type": "string",
			"enum": []any{"merge", "split", "keep"},
		},
		"why": map[string]any{"type": "string"},
	},
	"required": []any{"action", "why"},
}

// EntityLinkInput is context for a domain-shared entity pair.
type EntityLinkInput struct {
	EntityA      string
	EntityB      string
	SharedDomain string
	PainA        string
	PainB        string
	SightingA    int
	SightingB    int
}

// EntityLinkResult is Gemini recommendation for graph hygiene.
type EntityLinkResult struct {
	Action string
	Why    string
}

type entityLinkResponse struct {
	Action string `json:"action"`
	Why    string `json:"why"`
}

// ClassifyEntityLink recommends merge/split/keep for a conflicting entity pair.
func (c *Client) ClassifyEntityLink(ctx context.Context, in EntityLinkInput) (EntityLinkResult, error) {
	payload, err := json.Marshal(map[string]any{
		"entity_a":         strings.TrimSpace(in.EntityA),
		"entity_b":         strings.TrimSpace(in.EntityB),
		"shared_domain":    strings.TrimSpace(in.SharedDomain),
		"unified_pain_a":   strings.TrimSpace(in.PainA),
		"unified_pain_b":   strings.TrimSpace(in.PainB),
		"sighting_count_a": in.SightingA,
		"sighting_count_b": in.SightingB,
	})
	if err != nil {
		return EntityLinkResult{}, err
	}
	parsed, err := classifyJSON[entityLinkResponse](c, ctx, PriorityLow, entityLinkSystem,
		fmt.Sprintf("Entity link review:\n\n%s", payload), entityLinkSchema)
	if err != nil {
		return EntityLinkResult{}, err
	}
	action := strings.ToLower(strings.TrimSpace(parsed.Action))
	switch action {
	case "merge", "split", "keep":
	default:
		action = "keep"
	}
	return EntityLinkResult{
		Action: action,
		Why:    strings.TrimSpace(parsed.Why),
	}, nil
}
