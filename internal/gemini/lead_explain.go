package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

const leadExplainSystemPrompt = `You write one-line inbox triage summaries for sales reps reviewing affiliate tracker leads.
Use lead priority, matched keywords, entity heat, and outreach angle when present.
No PII. Max 25 words. Plain English.`

var leadExplainSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{
			"type":        "string",
			"description": "One line why this lead is hot or worth contact",
		},
	},
	"required": []any{"summary"},
}

type LeadExplainInput struct {
	HashID        string   `json:"hash_id"`
	Priority      string   `json:"priority"`
	Score         int      `json:"score"`
	Source        string   `json:"source"`
	Matched       []string `json:"matched,omitempty"`
	Snippet       string   `json:"snippet"`
	HeatTier      string   `json:"heat_tier,omitempty"`
	EntityProof   string   `json:"entity_proof,omitempty"`
	OutreachAngle string   `json:"outreach_angle,omitempty"`
	ICP           string   `json:"icp,omitempty"`
}

type leadExplainResponse struct {
	Summary string `json:"summary"`
}

func (c *Client) ExplainLead(ctx context.Context, in LeadExplainInput) (string, error) {
	if c == nil {
		return "", nil
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	prompt := "Summarize why this lead matters for inbox triage.\n\n" + string(payload)
	parsed, err := classifyJSON[leadExplainResponse](c, ctx, PriorityLow, leadExplainSystemPrompt, prompt, leadExplainSchema)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(parsed.Summary), nil
}
