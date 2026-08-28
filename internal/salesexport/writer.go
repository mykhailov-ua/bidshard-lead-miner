package salesexport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// JunkReportFromDoc maps a stored cold-path Gemini junk report to Russian sales JSON.
func JunkReportFromDoc(doc sink.JunkReportDoc) JunkReportRU {
	out := JunkReportRU{
		Title:                   "Отчёт Gemini: почему лиды отклонялись",
		PeriodFrom:              FormatTimeUTC(doc.PeriodFrom),
		PeriodTo:                FormatTimeUTC(doc.PeriodTo),
		SampleCount:             doc.SampleCount,
		Summary:                 doc.Summary,
		FalseNegativeCandidates: doc.FalseNegativeCandidates,
		Recommendations:         append([]string(nil), doc.Recommendations...),
		KeywordSuggestions:      append([]string(nil), doc.KeywordSuggestions...),
	}
	for _, r := range doc.TopReasons {
		out.TopReasons = append(out.TopReasons, ReasonRowRU{
			Reason: r.Reason,
			Count:  r.Count,
			Why:    r.Why,
		})
	}
	for _, s := range doc.SourceStats {
		out.SourceStats = append(out.SourceStats, SourceStatRU{
			Source: s.Source,
			Count:  s.Count,
		})
	}
	return out
}

// WriteJSON writes a sales export file under dir with dated name prefix.
func WriteJSON(dir, prefix string, v any) (string, error) {
	if dir == "" {
		dir = "data/export/sales"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102")
	path := filepath.Join(dir, prefix+"_"+stamp+".json")
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
