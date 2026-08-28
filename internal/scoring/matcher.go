package scoring

import (
	"sort"
	"strings"
)

type MatchHit struct {
	Phrase string `json:"phrase"`
	Weight int    `json:"weight"`
	Tag    string `json:"tag"`
}

type MatchResult struct {
	Score   int
	Hits    []MatchHit
	Summary []string
}

type span struct {
	start, end int
}

func matchWeighted(text string, rules []keywordRule) MatchResult {
	text = strings.ToLower(text)
	sorted := append([]keywordRule(nil), rules...)
	sort.Slice(sorted, func(i, j int) bool {
		if len(sorted[i].phrase) != len(sorted[j].phrase) {
			return len(sorted[i].phrase) > len(sorted[j].phrase)
		}
		return sorted[i].weight > sorted[j].weight
	})

	var claimed []span
	var hits []MatchHit
	total := 0

	for _, rule := range sorted {
		phrase := rule.phrase
		if phrase == "" {
			continue
		}
		start := 0
		for {
			idx := strings.Index(text[start:], phrase)
			if idx < 0 {
				break
			}
			absStart := start + idx
			absEnd := absStart + len(phrase)
			if !phraseMatchAt(text, absStart, rule.phrase) {
				start = absStart + 1
				continue
			}
			if !overlaps(claimed, absStart, absEnd) {
				claimed = append(claimed, span{absStart, absEnd})
				total += rule.weight
				hits = append(hits, MatchHit{
					Phrase: rule.phrase,
					Weight: rule.weight,
					Tag:    rule.tag,
				})
				break
			}
			start = absStart + 1
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Weight != hits[j].Weight {
			return hits[i].Weight > hits[j].Weight
		}
		return hits[i].Phrase < hits[j].Phrase
	})

	summary := make([]string, 0, len(hits))
	for _, h := range hits {
		summary = append(summary, formatHit(h))
	}

	return MatchResult{Score: total, Hits: hits, Summary: summary}
}

func overlaps(claimed []span, start, end int) bool {
	for _, c := range claimed {
		if start < c.end && end > c.start {
			return true
		}
	}
	return false
}

func formatHit(h MatchHit) string {
	if h.Weight < 0 {
		return h.Phrase + "(-" + itoa(-h.Weight) + ")"
	}
	return h.Phrase + "(+" + itoa(h.Weight) + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
