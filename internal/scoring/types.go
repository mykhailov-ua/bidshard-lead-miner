package scoring

type keywordRule struct {
	id     string
	phrase string
	weight int
	tag    string
}

type keywordEntry struct {
	ID     string `json:"id"`
	Phrase string `json:"phrase"`
	Weight int    `json:"weight"`
	Tag    string `json:"tag"`
}

type priorityConfig struct {
	HighMin   int `json:"high_min"`
	MediumMin int `json:"medium_min"`
}

const (
	defaultHighMin   = 35
	defaultMediumMin = 15
)

type keywordsFile struct {
	Priority    priorityConfig `json:"priority"`
	Keywords    []keywordEntry `json:"keywords"`
	Titles      []keywordEntry `json:"titles"`
	Negative    []keywordEntry `json:"negative"`
	HardReject  []keywordEntry `json:"hard_reject"`
}

func entryToRule(e keywordEntry) keywordRule {
	return keywordRule{id: e.ID, phrase: e.Phrase, weight: e.Weight, tag: e.Tag}
}

func rulesFromEntries(entries []keywordEntry) []keywordRule {
	out := make([]keywordRule, 0, len(entries))
	for _, e := range entries {
		if e.Phrase == "" {
			continue
		}
		out = append(out, entryToRule(e))
	}
	return out
}
