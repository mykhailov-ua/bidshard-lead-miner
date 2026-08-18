package gemini

import (
	"context"
	"strings"
)

const icpSystemPrompt = `Classify affiliate/iGaming lead candidates for BidShard tracker sales.
ICP starter: media buyers $15k+/mo spend, tracker pain, Voluum/Keitaro/RedTrack refugees.
ICP pro: OpenRTB/programmatic, platform engineers, small ad networks.
Return conservative labels; hot=true only for clear buyer pain with spend or competitor stack.`

var icpSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"icp": map[string]any{
			"type": "string",
			"enum": []any{"starter", "pro", "none"},
		},
		"hot":        map[string]any{"type": "boolean"},
		"spend_tier": map[string]any{
			"type": "string",
			"enum": []any{"15k-150k", "unknown"},
		},
		"why": map[string]any{"type": "string"},
	},
	"required": []any{"icp", "hot", "spend_tier"},
}

type ICPResult struct {
	ICP       string
	Hot       bool
	SpendTier string
	Why       string
}

type icpResponse struct {
	ICP       string `json:"icp"`
	Hot       bool   `json:"hot"`
	SpendTier string `json:"spend_tier"`
	Why       string `json:"why"`
}

func (c *Client) ClassifyICP(ctx context.Context, text string) (ICPResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return ICPResult{ICP: "none", SpendTier: "unknown"}, nil
	}
	raw, err := c.generateJSON(ctx, icpSystemPrompt, "Classify this lead snippet:\n\n"+truncate(text, 2000), icpSchema)
	if err != nil {
		return ICPResult{}, err
	}
	var parsed icpResponse
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return ICPResult{}, err
	}
	return ICPResult{
		ICP:       normalizeICP(parsed.ICP),
		Hot:       parsed.Hot,
		SpendTier: normalizeSpendTier(parsed.SpendTier),
		Why:       strings.TrimSpace(parsed.Why),
	}, nil
}

func normalizeICP(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "starter", "pro":
		return strings.ToLower(v)
	default:
		return "none"
	}
}

func normalizeSpendTier(v string) string {
	if strings.EqualFold(v, "15k-150k") {
		return "15k-150k"
	}
	return "unknown"
}

func ApplyICPToScore(score int, icp ICPResult, highMin int) (int, string) {
	priority := ""
	switch {
	case icp.Hot && icp.ICP != "none":
		score += 12
	case icp.ICP == "none" && !icp.Hot:
		if score > 0 {
			score -= 8
		}
	}
	if score >= highMin {
		priority = "High"
	}
	return score, priority
}
