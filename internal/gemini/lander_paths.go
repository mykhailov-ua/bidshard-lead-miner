package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

const landerPathRankSystemPrompt = `You rank website paths for affiliate-network LPR contact discovery.
Prefer pages likely to list partnerships@media-buyer contacts: affiliate program, partners, contact, about.
Drop blog, news, login, careers, and product docs paths.
Return at most 8 paths.`

var landerPathRankSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"paths": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required": []any{"paths"},
}

type landerPathRankResponse struct {
	Paths []string `json:"paths"`
}

// RankLanderPaths narrows candidate paths for tgweb/lander crawl (implements lander.PathRanker).
func (c *Client) RankLanderPaths(ctx context.Context, domain string, candidates []string) ([]string, error) {
	if c == nil || len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) <= 8 {
		return candidates, nil
	}
	payload, err := json.Marshal(map[string]any{
		"domain":     domain,
		"candidates": candidates,
	})
	if err != nil {
		return nil, err
	}
	prompt := "Rank paths most likely to expose affiliate/partner contact emails.\n\n" + string(payload)
	parsed, err := classifyJSON[landerPathRankResponse](c, ctx, PriorityLow, landerPathRankSystemPrompt, prompt, landerPathRankSchema)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(parsed.Paths))
	seen := map[string]struct{}{}
	for _, p := range parsed.Paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
		if len(out) >= 8 {
			break
		}
	}
	return out, nil
}
