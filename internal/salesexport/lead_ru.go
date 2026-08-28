package salesexport

import (
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/sink"
)

var pilotSignalRU = map[string]string{
	"spend_budget":     "бюджет / высокие траты",
	"competitor_stack": "использует конкурентный трекер",
	"tracker_pain":     "боль с трекером / постбеками",
	"infra_vps":        "своя инфраструктура / VPS",
	"usdt_ok":          "готов платить в USDT",
	"buyer_role":       "лицо принимающее решение",
	"high_volume":      "высокие объёмы трафика",
	"migration_intent": "намерение мигрировать",
}

var companyTypeRU = map[string]string{
	"media_buyer":       "медиабаер",
	"affiliate_network": "партнёрская сеть",
	"tool_vendor":       "вендор инструментов",
	"agency":            "агентство",
	"unknown":           "неизвестно",
}

var outreachChannelRU = map[string]string{
	"telegram": "Telegram",
	"email":    "Email",
	"forum":    "форум (ручной ответ)",
	"other":    "другое",
}

var outcomeRU = map[string]string{
	"contacted":          "связались",
	"replied":            "ответил",
	"pilot_started":      "пилот запущен",
	"migration_imported": "миграция завершена",
}

var priorityRU = map[string]string{
	"high":   "высокий",
	"medium": "средний",
	"low":    "низкий",
}

var heatTierRU = map[string]string{
	"blazing": "горячий",
	"hot":     "тёплый",
	"warm":    "умеренный",
	"cold":    "холодный",
}

var nextActionRU = map[string]string{
	"telegram_dm":      "написать в Telegram",
	"cold_email":       "отправить cold email",
	"skype_message":    "написать в Skype",
	"forum_manual":     "ответить на форуме вручную",
	"reddit_dm":        "написать в Reddit",
	"github_reach":     "связаться через GitHub",
	"research_contact": "найти контакт",
}

// LeadCardFromDoc maps a Mongo lead into a Russian sales card (static labels).
func LeadCardFromDoc(doc sink.LeadDoc) LeadCardRU {
	contacts := make([]string, 0, len(doc.Contacts))
	for _, c := range doc.Contacts {
		line := strings.TrimSpace(c.Type + ":" + c.Value)
		if line == ":" {
			continue
		}
		contacts = append(contacts, line)
	}
	signals := make([]string, 0)
	for _, tag := range doc.Tags {
		if ru, ok := pilotSignalRU[tag]; ok {
			signals = append(signals, ru)
		}
	}
	return LeadCardRU{
		HashID:          doc.HashID,
		Priority:        translate(priorityRU, doc.Priority, doc.Priority),
		Score:           doc.Score,
		HeatTier:        translate(heatTierRU, doc.HeatTier, doc.HeatTier),
		Source:          doc.Source,
		PostedAt:        FormatTimeUTC(doc.PostedAt),
		Contacts:        contacts,
		MatchedKeywords: append([]string(nil), doc.Matched...),
		ContextSnippet:  doc.Snippet,
		ICP:             doc.ICP,
		ICPExplanation:  doc.ICPWhy,
		GeoCountry:      doc.GeoCountry,
		CompanyType:     translate(companyTypeRU, doc.CompanyType, doc.CompanyType),
		EnrichSummary:   doc.EnrichSummary,
		PilotQualified:  doc.PilotQualified,
		PilotWhy:        doc.PilotWhy,
		PilotSignals:    signals,
		OutreachChannel: translate(outreachChannelRU, doc.OutreachChannel, doc.OutreachChannel),
		NextAction:      translate(nextActionRU, doc.NextAction, doc.NextAction),
		OutreachSubject: doc.OutreachSubject,
		OutreachAngle:   doc.OutreachAngle,
		OutreachDraft:   doc.OutreachDraft,
		EntityProof:     doc.EntityProof,
		Outcome:         translate(outcomeRU, doc.Outcome, doc.Outcome),
		OutcomeNote:     doc.OutcomeNote,
		Tags:            append([]string(nil), doc.Tags...),
	}
}

func translate(table map[string]string, key, fallback string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if v, ok := table[key]; ok && v != "" {
		return v
	}
	return strings.TrimSpace(fallback)
}

// FeedbackBundleFromDiscover builds a Russian feedback summary without Gemini.
func FeedbackBundleFromDiscover(dorks []discover.OutcomeDorkRow, keywords []discover.KeywordTuneRow, generatedAt string) FeedbackBundleRU {
	out := FeedbackBundleRU{
		Title:       "Отчёт обратной связи: источники и ключевые фразы",
		GeneratedAt: generatedAt,
	}
	for _, row := range dorks {
		out.DorkRows = append(out.DorkRows, DorkOutcomeRowRU{
			Query:            row.Query,
			Accepted:         row.Accepted,
			Junk:             row.Junk,
			AcceptRatePct:    roundPct(row.AcceptRate),
			OutcomeContacted: row.OutcomeContacted,
			OutcomeReplied:   row.OutcomeReplied,
			OutcomePilot:     row.OutcomePilot,
			OutcomeMigration: row.OutcomeMigration,
		})
	}
	for _, row := range keywords {
		out.KeywordRows = append(out.KeywordRows, KeywordTuneRowRU{
			KeywordID:       row.KeywordID,
			Accepted:        row.Accepted,
			Junk:            row.Junk,
			JunkRatePct:     roundPct(row.JunkRate),
			SuggestedWeight: row.SuggestedWeight,
			RecommendOff:    row.RecommendDisable,
		})
	}
	return out
}

func roundPct(v float64) float64 {
	return float64(int(v*10000+0.5)) / 100
}
