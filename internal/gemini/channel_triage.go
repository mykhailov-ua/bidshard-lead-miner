package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

const channelTriageSystemPrompt = `You triage Telegram channels for an affiliate tracker lead parser.
Drop channels that are CIS/geo noise, off-topic, or not media-buyer relevant.
Keep channels likely to surface Voluum/tracker/postback buyer pain.`

var channelTriageSchema = map[string]any{
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
						"enum": []any{"keep", "drop"},
					},
					"why": map[string]any{"type": "string"},
				},
				"required": []any{"id", "action"},
			},
		},
	},
	"required": []any{"items"},
}

type ChannelTriageInput struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Query string `json:"query,omitempty"`
	About string `json:"about,omitempty"`
}

type ChannelTriageResult struct {
	ID     string
	Action string
	Why    string
}

type channelTriageResponse struct {
	Items []struct {
		ID     string `json:"id"`
		Action string `json:"action"`
		Why    string `json:"why"`
	} `json:"items"`
}

func (c *Client) TriageChannels(ctx context.Context, items []ChannelTriageInput) ([]ChannelTriageResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return nil, err
	}
	prompt := "Triage each channel id. Return one item per input id.\n\n" + string(payload)
	parsed, err := classifyJSON[channelTriageResponse](c, ctx, PriorityLow, channelTriageSystemPrompt, prompt, channelTriageSchema)
	if err != nil {
		return nil, err
	}
	out := make([]ChannelTriageResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		action := strings.ToLower(strings.TrimSpace(item.Action))
		if action != "keep" && action != "drop" {
			action = "keep"
		}
		out = append(out, ChannelTriageResult{
			ID:     strings.TrimSpace(item.ID),
			Action: action,
			Why:    strings.TrimSpace(item.Why),
		})
	}
	return out, nil
}
