package gemini

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bidshard/parser/internal/discover"
)

const discoverDiffSystem = `You propose discovery query additions for a lead parser hunting affiliate/iGaming media buyers.
Output JSON only. Suggest new telegram_search phrases (plain text, no site: operator) and serp_dorks (Google dorks, often site:t.me).
Focus on tracker pain, migration, competitor alternatives, decision-maker channels.
Do not duplicate existing queries. Max 8 per list.`

var discoverDiffSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"add_telegram_search": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"add_serp_dorks": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"summary": map[string]any{"type": "string"},
	},
	"required": []any{"add_telegram_search", "add_serp_dorks", "summary"},
}

type DiscoverICPDiff struct {
	AddTelegramSearch []string `json:"add_telegram_search"`
	AddSerpDorks      []string `json:"add_serp_dorks"`
	Summary           string   `json:"summary"`
	ReportID          string   `json:"report_id,omitempty"`
	GeneratedAt       string   `json:"generated_at,omitempty"`
	Status            string   `json:"status"`
}

func (c *Client) BuildDiscoverICPDiff(
	ctx context.Context,
	current discover.ICPConfig,
	keywordSuggestions []string,
	falseNegativeSnippets []string,
) (DiscoverICPDiff, error) {
	payload, err := json.Marshal(map[string]any{
		"current_telegram_search": current.TelegramSearch,
		"current_serp_dorks":      current.SerpDorks,
		"keyword_suggestions":     keywordSuggestions,
		"false_negative_snippets": falseNegativeSnippets,
	})
	if err != nil {
		return DiscoverICPDiff{}, err
	}
	diff, err := classifyJSON[DiscoverICPDiff](c, ctx, PriorityLow, discoverDiffSystem,
		"Propose discover.icp.json additions for manual approve:\n\n"+string(payload),
		discoverDiffSchema)
	if err != nil {
		return DiscoverICPDiff{}, err
	}
	diff.Status = "pending"
	diff.Summary = strings.TrimSpace(diff.Summary)
	diff.AddTelegramSearch = discover.UniqueFolded(nil, diff.AddTelegramSearch)
	diff.AddSerpDorks = discover.UniqueFolded(nil, diff.AddSerpDorks)
	return diff, nil
}
