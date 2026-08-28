package gemini

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bidshard/parser/internal/pretty"
)

const intentSystemPrompt = `Classify short affiliate/media-buying chat snippets for a self-hosted ad tracker parser.
Return intent and confidence for each item. Outreach only wants active buyers searching for tracker solutions.

intent values:
- buyer_search: choosing/switching trackers, asking for alternatives, evaluating spend tools
- technical_issue: tracker/postback/cloak bug while already using a stack (may be buyer but not search)
- news: announcements, product releases, industry news without personal buyer need
- job_offer: hiring, recruiting, job posts, course sellers
- noise: spam, off-topic, memes, generic affiliate promo without buyer voice

Be strict: buyer_search requires first-person buyer voice or direct tool-selection question.`

var intentSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"intent": map[string]any{
			"type": "string",
			"enum": []any{"buyer_search", "technical_issue", "news", "job_offer", "noise"},
		},
		"confidence": map[string]any{
			"type":    "number",
			"minimum": 0,
			"maximum": 1,
		},
		"why": map[string]any{"type": "string"},
	},
	"required": []any{"intent", "confidence"},
}

type IntentResult struct {
	Intent     string
	Confidence float64
	Why        string
}

type intentResponse struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Why        string  `json:"why"`
}

// Accept reports whether the result passes outreach intent gate at min confidence.
func (r IntentResult) Accept(minConfidence float64) bool {
	if minConfidence <= 0 {
		minConfidence = 0.8
	}
	return normalizeIntent(r.Intent) == "buyer_search" && r.Confidence >= minConfidence
}

func (c *Client) ClassifyIntent(ctx context.Context, text string) (IntentResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return IntentResult{Intent: "noise", Confidence: 1}, nil
	}
	prompt := "Classify intent:\n\n" + pretty.Truncate(text, 2000)
	parsed, err := classifyJSON[intentResponse](c, ctx, PriorityNormal, intentSystemPrompt, prompt, intentSchema)
	if err != nil {
		return IntentResult{}, err
	}
	return IntentResult{
		Intent:     normalizeIntent(parsed.Intent),
		Confidence: clampConfidence(parsed.Confidence),
		Why:        strings.TrimSpace(parsed.Why),
	}, nil
}

type IntentBatchInput struct {
	ID     string
	Source string
	Text   string
}

type IntentBatchResult struct {
	ID         string
	Intent     string
	Confidence float64
	Why        string
}

var intentBatchSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
					"intent": map[string]any{
						"type": "string",
						"enum": []any{"buyer_search", "technical_issue", "news", "job_offer", "noise"},
					},
					"confidence": map[string]any{
						"type":    "number",
						"minimum": 0,
						"maximum": 1,
					},
					"why": map[string]any{"type": "string"},
				},
				"required": []any{"id", "intent", "confidence"},
			},
		},
	},
	"required": []any{"items"},
}

type intentBatchResponse struct {
	Items []struct {
		ID         string  `json:"id"`
		Intent     string  `json:"intent"`
		Confidence float64 `json:"confidence"`
		Why        string  `json:"why"`
	} `json:"items"`
}

// AnalyzeIntentBatch classifies multiple snippets (cold-path or buffered hot-path batches).
func (c *Client) AnalyzeIntentBatch(ctx context.Context, items []IntentBatchInput) ([]IntentBatchResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	payload, err := marshalIntentBatchPayload(items)
	if err != nil {
		return nil, err
	}
	prompt := "Classify each item id. Return one row per input id.\n\n" + string(payload)
	parsed, err := classifyJSON[intentBatchResponse](c, ctx, PriorityNormal, intentSystemPrompt, prompt, intentBatchSchema)
	if err != nil {
		return nil, err
	}
	out := make([]IntentBatchResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, IntentBatchResult{
			ID:         strings.TrimSpace(item.ID),
			Intent:     normalizeIntent(item.Intent),
			Confidence: clampConfidence(item.Confidence),
			Why:        strings.TrimSpace(item.Why),
		})
	}
	return out, nil
}

func marshalIntentBatchPayload(items []IntentBatchInput) ([]byte, error) {
	type row struct {
		ID     string `json:"id"`
		Source string `json:"source,omitempty"`
		Text   string `json:"text"`
	}
	rows := make([]row, 0, len(items))
	for _, item := range items {
		rows = append(rows, row{
			ID:     item.ID,
			Source: item.Source,
			Text:   pretty.Truncate(strings.TrimSpace(item.Text), 1500),
		})
	}
	return json.Marshal(map[string]any{"items": rows})
}

func normalizeIntent(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "buyer_search", "technical_issue", "news", "job_offer", "noise":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "noise"
	}
}

func clampConfidence(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
