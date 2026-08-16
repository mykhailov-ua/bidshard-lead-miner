package scoring

import "strings"

type Priority string

const (
	PriorityHigh   Priority = "High"
	PriorityMedium Priority = "Medium"
	PriorityLow    Priority = "Low"
)

type LeadText struct {
	Title   string
	Context string
	Score   int
	Matched []string
}

func ScoreText(reg *Registry, text *LeadText) Priority {
	result := AnalyzeWithRegistry(reg, text.Context+" "+text.Title)
	text.Score = result.Score
	text.Matched = result.Summary

	_, _, _, highMin, mediumMin := reg.Snapshot()
	switch {
	case result.Score >= highMin:
		return PriorityHigh
	case result.Score >= mediumMin:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

func AnalyzeWithRegistry(reg *Registry, text string) MatchResult {
	body := strings.ToLower(text)
	kwRules, titleRules, negRules, _, _ := reg.Snapshot()

	kw := matchWeighted(body, kwRules)
	title := matchWeighted(body, titleRules)
	neg := matchWeighted(body, negRules)

	return MatchResult{
		Score:   kw.Score + title.Score + neg.Score,
		Hits:    append(append(kw.Hits, title.Hits...), neg.Hits...),
		Summary: append(append(kw.Summary, title.Summary...), neg.Summary...),
	}
}

func Analyze(text string) MatchResult {
	return AnalyzeWithRegistry(defaultRegistry, text)
}

var defaultRegistry = NewRegistry("data/keywords.json")

func SetRegistry(r *Registry) {
	if r != nil {
		defaultRegistry = r
	}
}
