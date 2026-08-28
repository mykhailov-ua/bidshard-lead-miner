package gemini

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bidshard/parser/internal/salesexport"
	"github.com/bidshard/parser/internal/sink"
)

const salesJunkReportRUSystem = `Ты готовишь JSON-отчёт для русскоязычного сейлс-менеджера (не программист).
Переведи и перефразируй тексты на понятный русский. Сохрани факты, не выдумывай PII.
Пиши кратко, деловым языком, без англицизмов где есть русский аналог.`

var salesJunkReportRUSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"краткое_резюме": map[string]any{"type": "string"},
		"топ_причин": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"причина":    map[string]any{"type": "string"},
					"количество": map[string]any{"type": "integer"},
					"пояснение":  map[string]any{"type": "string"},
				},
				"required": []any{"причина", "количество"},
			},
		},
		"рекомендации": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"идеи_ключевых_фраз": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required": []any{"краткое_резюме", "топ_причин", "рекомендации"},
}

type salesJunkReportRUResponse struct {
	Summary    string `json:"краткое_резюме"`
	TopReasons []struct {
		Reason string `json:"причина"`
		Count  int    `json:"количество"`
		Why    string `json:"пояснение"`
	} `json:"топ_причин"`
	Recommendations    []string `json:"рекомендации"`
	KeywordSuggestions []string `json:"идеи_ключевых_фраз"`
}

// LocalizeJunkReportRU rewrites a junk tuning report in Russian for sales/ops review.
func (c *Client) LocalizeJunkReportRU(ctx context.Context, doc sink.JunkReportDoc) (salesexport.JunkReportRU, error) {
	base := salesexport.JunkReportFromDoc(doc)
	if c == nil {
		return base, nil
	}
	payload, err := json.Marshal(map[string]any{
		"period_from":               base.PeriodFrom,
		"period_to":                 base.PeriodTo,
		"sample_count":              base.SampleCount,
		"summary_en":                doc.Summary,
		"top_reasons":               doc.TopReasons,
		"recommendations_en":        doc.Recommendations,
		"keyword_suggestions_en":    doc.KeywordSuggestions,
		"false_negative_candidates": doc.FalseNegativeCandidates,
	})
	if err != nil {
		return base, err
	}
	parsed, err := classifyJSON[salesJunkReportRUResponse](c, ctx, PriorityLow, salesJunkReportRUSystem,
		"Переведи отчёт по отклонённым лидам для менеджера по продажам:\n\n"+string(payload), salesJunkReportRUSchema)
	if err != nil {
		return base, err
	}
	out := base
	out.Summary = strings.TrimSpace(parsed.Summary)
	out.Recommendations = append([]string(nil), parsed.Recommendations...)
	out.KeywordSuggestions = append([]string(nil), parsed.KeywordSuggestions...)
	out.TopReasons = nil
	for _, r := range parsed.TopReasons {
		out.TopReasons = append(out.TopReasons, salesexport.ReasonRowRU{
			Reason: r.Reason,
			Count:  r.Count,
			Why:    strings.TrimSpace(r.Why),
		})
	}
	return out, nil
}

const salesLeadsRUSystem = `Ты готовишь карточки лидов для русскоязычного сейлс-менеджера.
Для каждого лида переведи на русский: почему_пилот, угол_подхода, текст_сообщения, профиль_клиента, доказательства_сущности.
Не выдумывай контакты и имена. Если поле пустое - оставь пустую строку.`

var salesLeadsRUSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"лиды": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id_лида":                 map[string]any{"type": "string"},
					"почему_пилот":            map[string]any{"type": "string"},
					"угол_подхода":            map[string]any{"type": "string"},
					"текст_сообщения":         map[string]any{"type": "string"},
					"профиль_клиента":         map[string]any{"type": "string"},
					"доказательства_сущности": map[string]any{"type": "string"},
					"тема_письма":             map[string]any{"type": "string"},
				},
				"required": []any{"id_лида"},
			},
		},
	},
	"required": []any{"лиды"},
}

type salesLeadsRUResponse struct {
	Leads []struct {
		HashID          string `json:"id_лида"`
		PilotWhy        string `json:"почему_пилот"`
		OutreachAngle   string `json:"угол_подхода"`
		OutreachDraft   string `json:"текст_сообщения"`
		EnrichSummary   string `json:"профиль_клиента"`
		EntityProof     string `json:"доказательства_сущности"`
		OutreachSubject string `json:"тема_письма"`
	} `json:"лиды"`
}

// LocalizeLeadCardsRU translates Gemini narrative fields on lead cards to Russian.
func (c *Client) LocalizeLeadCardsRU(ctx context.Context, cards []salesexport.LeadCardRU) ([]salesexport.LeadCardRU, error) {
	if len(cards) == 0 {
		return nil, nil
	}
	if c == nil {
		return cards, nil
	}
	payload, err := json.Marshal(map[string]any{"leads": cards})
	if err != nil {
		return cards, err
	}
	parsed, err := classifyJSON[salesLeadsRUResponse](c, ctx, PriorityLow, salesLeadsRUSystem,
		"Переведи поля Gemini для карточек лидов:\n\n"+string(payload), salesLeadsRUSchema)
	if err != nil {
		return cards, err
	}
	byID := map[string]int{}
	for i, card := range cards {
		byID[strings.TrimSpace(card.HashID)] = i
	}
	for _, row := range parsed.Leads {
		idx, ok := byID[strings.TrimSpace(row.HashID)]
		if !ok {
			continue
		}
		if v := strings.TrimSpace(row.PilotWhy); v != "" {
			cards[idx].PilotWhy = v
		}
		if v := strings.TrimSpace(row.OutreachAngle); v != "" {
			cards[idx].OutreachAngle = v
		}
		if v := strings.TrimSpace(row.OutreachDraft); v != "" {
			cards[idx].OutreachDraft = v
		}
		if v := strings.TrimSpace(row.EnrichSummary); v != "" {
			cards[idx].EnrichSummary = v
		}
		if v := strings.TrimSpace(row.EntityProof); v != "" {
			cards[idx].EntityProof = v
		}
		if v := strings.TrimSpace(row.OutreachSubject); v != "" {
			cards[idx].OutreachSubject = v
		}
	}
	return cards, nil
}
