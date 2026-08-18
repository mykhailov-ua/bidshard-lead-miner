package gemini

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

const analyzerSystemPrompt = `You analyze rejected affiliate/iGaming lead candidates for a self-hosted ad tracker (BidShard).
Audience: media buyers outside Russia/Belarus looking for tracker alternatives (Voluum, Keitaro, RedTrack, etc.).
Contacts in input are already masked — never ask for or invent PII.
Classify each item:
- true_junk: spam, off-topic, newbie with no spend, job seeker, course seller, white AdTech, RU/BY geo signals
- false_negative: likely real buyer pain we should not have dropped (tune keywords/scoring/geo)
- borderline: weak signal but not clearly junk

Be concise. Focus on filter tuning, not sales copy.`

var analyzeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Same id from input",
					},
					"category": map[string]any{
						"type": "string",
						"enum": []any{"true_junk", "false_negative", "borderline"},
					},
					"why": map[string]any{
						"type":        "string",
						"description": "1-2 sentences why this category",
					},
					"suggestions": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required": []any{"id", "category", "why"},
			},
		},
	},
	"required": []any{"items"},
}

type analyzeInputItem struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Reason       string   `json:"reason"`
	ReasonDetail string   `json:"reason_detail,omitempty"`
	Score        int      `json:"score,omitempty"`
	Matched      []string `json:"matched,omitempty"`
	Snippet      string   `json:"snippet"`
}

type AnalyzeResult struct {
	ID          string
	Category    string
	Why         string
	Suggestions []string
}

type analyzeResponse struct {
	Items []struct {
		ID          string   `json:"id"`
		Category    string   `json:"category"`
		Why         string   `json:"why"`
		Suggestions []string `json:"suggestions"`
	} `json:"items"`
}

func (c *Client) AnalyzeJunkBatch(ctx context.Context, docs []sink.JunkDoc) ([]AnalyzeResult, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	items := make([]analyzeInputItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, analyzeInputItem{
			ID:           doc.ID.Hex(),
			Source:       doc.Source,
			Reason:       doc.Reason,
			ReasonDetail: doc.ReasonDetail,
			Score:        doc.Score,
			Matched:      doc.Matched,
			Snippet:      doc.Snippet,
		})
	}

	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return nil, err
	}

	raw, err := c.generateJSON(ctx, analyzerSystemPrompt,
		"Analyze each rejected lead. Return one output item per input id.\n\n"+string(payload),
		analyzeSchema)
	if err != nil {
		return nil, err
	}

	var parsed analyzeResponse
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return nil, err
	}

	out := make([]AnalyzeResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, AnalyzeResult{
			ID:          strings.TrimSpace(item.ID),
			Category:    normalizeCategory(item.Category),
			Why:         strings.TrimSpace(item.Why),
			Suggestions: item.Suggestions,
		})
	}
	return out, nil
}

func normalizeCategory(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false_negative", "borderline", "true_junk":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "borderline"
	}
}

const reporterSystemPrompt = `You write operational reports for a lead parser tuning team.
Summarize why leads were rejected, whether filters are too aggressive, and concrete changes (keywords, geo rules, score thresholds).
Do not include raw contacts. Be specific and actionable.`

var reportSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{
			"type":        "string",
			"description": "Executive summary, 2-4 sentences",
		},
		"top_reasons": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"reason": map[string]any{"type": "string"},
					"count":  map[string]any{"type": "integer"},
					"why":    map[string]any{"type": "string"},
				},
				"required": []any{"reason", "count", "why"},
			},
		},
		"false_negative_candidates": map[string]any{"type": "integer"},
		"recommendations": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"keyword_suggestions": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "New keyword phrases to add or negative phrases to hard_reject",
		},
	},
	"required": []any{"summary", "top_reasons", "false_negative_candidates", "recommendations"},
}

type ReportInput struct {
	PeriodFrom  string              `json:"period_from"`
	PeriodTo    string              `json:"period_to"`
	TotalJunk   int64               `json:"total_junk"`
	ReasonStats []sink.ReasonCount  `json:"reason_stats"`
	Samples     []reportSampleInput `json:"samples"`
}

type reportSampleInput struct {
	Reason       string `json:"reason"`
	ReasonDetail string `json:"reason_detail,omitempty"`
	Score        int    `json:"score,omitempty"`
	Category     string `json:"category,omitempty"`
	Why          string `json:"why,omitempty"`
	Snippet      string `json:"snippet"`
}

type ReportResult struct {
	Summary                 string
	TopReasons              []sink.ReasonCount
	FalseNegativeCandidates int
	Recommendations         []string
	KeywordSuggestions      []string
}

type reportResponse struct {
	Summary    string `json:"summary"`
	TopReasons []struct {
		Reason string `json:"reason"`
		Count  int    `json:"count"`
		Why    string `json:"why"`
	} `json:"top_reasons"`
	FalseNegativeCandidates int      `json:"false_negative_candidates"`
	Recommendations         []string `json:"recommendations"`
	KeywordSuggestions      []string `json:"keyword_suggestions"`
}

func (c *Client) BuildJunkReport(ctx context.Context, in ReportInput) (ReportResult, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return ReportResult{}, err
	}

	raw, err := c.generateJSON(ctx, reporterSystemPrompt,
		"Write a tuning report for this rejection window.\n\n"+string(payload),
		reportSchema)
	if err != nil {
		return ReportResult{}, err
	}

	var parsed reportResponse
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return ReportResult{}, err
	}

	top := make([]sink.ReasonCount, 0, len(parsed.TopReasons))
	for _, r := range parsed.TopReasons {
		top = append(top, sink.ReasonCount{
			Reason: r.Reason,
			Count:  r.Count,
			Why:    r.Why,
		})
	}

	return ReportResult{
		Summary:                 strings.TrimSpace(parsed.Summary),
		TopReasons:              top,
		FalseNegativeCandidates: parsed.FalseNegativeCandidates,
		Recommendations:         parsed.Recommendations,
		KeywordSuggestions:      parsed.KeywordSuggestions,
	}, nil
}

func ReportInputFromStore(since, until time.Time, total int64, stats []sink.ReasonCount, samples []sink.JunkDoc) ReportInput {
	in := ReportInput{
		PeriodFrom:  since.UTC().Format(time.RFC3339),
		PeriodTo:    until.UTC().Format(time.RFC3339),
		TotalJunk:   total,
		ReasonStats: stats,
	}
	for _, s := range samples {
		item := reportSampleInput{
			Reason:       s.Reason,
			ReasonDetail: s.ReasonDetail,
			Score:        s.Score,
			Snippet:      s.Snippet,
		}
		if s.Analysis != nil {
			item.Category = s.Analysis.Category
			item.Why = s.Analysis.Why
		}
		in.Samples = append(in.Samples, item)
	}
	return in
}
