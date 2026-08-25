package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

const domainTriageSystemPrompt = `You triage affiliate-site domains before HTTP crawl for a media-buyer lead parser.
Drop domains that are obvious noise: social networks, URL shorteners, search engines, unrelated blogs, or CIS/geo junk.
Keep domains likely to host affiliate programs, trackers, or buyer pain (Voluum/postback/igaming affiliate).
Use defer only when metadata is too thin to decide.`

var domainTriageSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
					"action": map[string]any{
						"type": "string",
						"enum": []any{"keep", "drop", "defer"},
					},
					"why": map[string]any{"type": "string"},
				},
				"required": []any{"id", "action"},
			},
		},
	},
	"required": []any{"items"},
}

type DomainTriageInput struct {
	ID           string `json:"id"`
	Domain       string `json:"domain"`
	Channel      string `json:"channel,omitempty"`
	Source       string `json:"source,omitempty"`
	DiscoveredBy string `json:"discovered_by,omitempty"`
	Kind         string `json:"kind,omitempty"`
}

type DomainTriageResult struct {
	ID     string
	Action string
	Why    string
}

type domainTriageResponse struct {
	Items []struct {
		ID     string `json:"id"`
		Action string `json:"action"`
		Why    string `json:"why"`
	} `json:"items"`
}

func (c *Client) TriageDomains(ctx context.Context, items []DomainTriageInput) ([]DomainTriageResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return nil, err
	}
	prompt := "Triage each domain id. Return one item per input id.\n\n" + string(payload)
	parsed, err := classifyJSON[domainTriageResponse](c, ctx, PriorityLow, domainTriageSystemPrompt, prompt, domainTriageSchema)
	if err != nil {
		return nil, err
	}
	out := make([]DomainTriageResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		action := strings.ToLower(strings.TrimSpace(item.Action))
		switch action {
		case "keep", "drop", "defer":
		default:
			action = "defer"
		}
		out = append(out, DomainTriageResult{
			ID:     strings.TrimSpace(item.ID),
			Action: action,
			Why:    strings.TrimSpace(item.Why),
		})
	}
	return out, nil
}
