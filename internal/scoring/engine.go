package scoring

import (
	"strings"
	"time"
)

type Priority string

const (
	PriorityHigh   Priority = "High"
	PriorityMedium Priority = "Medium"
	PriorityLow    Priority = "Low"
)

func MeetsMinPriority(p, min Priority) bool {
	rank := map[Priority]int{
		PriorityLow:    0,
		PriorityMedium: 1,
		PriorityHigh:   2,
	}
	return rank[p] >= rank[min]
}

type LeadText struct {
	Title   string
	Context string
	Score   int
	Matched []string
}

func ScoreWithBoosts(reg *Registry, text *LeadText, source string, stack []string, rep *SourceReputation, opts ScoreOpts) Priority {
	combined := text.Context + " " + text.Title
	result := AnalyzeWithRegistry(reg, combined)
	_, _, _, highMin, mediumMin := reg.Snapshot()
	score := ApplySpendGate(result.Score, combined, mediumMin)
	score = CompetitorPainBoost(score, combined, stack)
	if rep != nil {
		score += rep.Boost(source)
	}
	if opts.TimeDecay {
		score = ApplyTimeDecay(score, opts.PostedAt, time.Now().UTC())
	}
	text.Score = score
	text.Matched = result.Summary

	switch {
	case text.Score >= highMin:
		return PriorityHigh
	case text.Score >= mediumMin:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

func ScoreText(reg *Registry, text *LeadText) Priority {
	return ScoreWithBoosts(reg, text, "", nil, nil, ScoreOpts{})
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
