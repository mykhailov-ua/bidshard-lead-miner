package gemini

import (
	"context"
	"encoding/json"
	"strings"
)

const keywordDiffSystem = `You propose keyword registry changes for a lead parser.
Output JSON only. Suggest additions to keywords (positive intent/pain) or hard_reject (anti-ICP).
Do not remove existing keywords. Weight 15-25 for intent/pain phrases.`

var keywordDiffSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"add_keywords": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"phrase": map[string]any{"type": "string"},
					"weight": map[string]any{"type": "integer"},
					"tag":    map[string]any{"type": "string"},
				},
				"required": []any{"id", "phrase", "weight", "tag"},
			},
		},
		"add_hard_reject": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"phrase": map[string]any{"type": "string"},
					"tag":    map[string]any{"type": "string"},
				},
				"required": []any{"id", "phrase"},
			},
		},
		"summary": map[string]any{"type": "string"},
	},
	"required": []any{"add_keywords", "add_hard_reject", "summary"},
}

type KeywordEntry struct {
	ID              string  `json:"id"`
	Phrase          string  `json:"phrase"`
	Weight          int     `json:"weight,omitempty"`
	Tag             string  `json:"tag,omitempty"`
	SuggestedWeight int     `json:"suggested_weight,omitempty"`
	EvidenceCount   int     `json:"evidence_count,omitempty"`
	JunkRate        float64 `json:"junk_rate,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

type KeywordDiff struct {
	AddKeywords   []KeywordEntry `json:"add_keywords"`
	AddHardReject []KeywordEntry `json:"add_hard_reject"`
	Summary       string         `json:"summary"`
	ReportID      string         `json:"report_id,omitempty"`
	GeneratedAt   string         `json:"generated_at,omitempty"`
	Status        string         `json:"status"` // pending
}

func (c *Client) BuildKeywordDiff(ctx context.Context, suggestions []string, falseNegativeSamples []string) (KeywordDiff, error) {
	payload, err := json.Marshal(map[string]any{
		"keyword_suggestions":     suggestions,
		"false_negative_snippets": falseNegativeSamples,
	})
	if err != nil {
		return KeywordDiff{}, err
	}
	diff, err := classifyJSON[KeywordDiff](c, ctx, PriorityLow, keywordDiffSystem,
		"Propose keyword diff for manual approve:\n\n"+string(payload), keywordDiffSchema)
	if err != nil {
		return KeywordDiff{}, err
	}
	diff.Status = "pending"
	diff.Summary = strings.TrimSpace(diff.Summary)
	return diff, nil
}
