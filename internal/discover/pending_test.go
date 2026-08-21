package discover

import "testing"

func TestMergeSuggestionsDedupes(t *testing.T) {
	t.Parallel()

	current := ICPConfig{
		TelegramSearch: []string{"Voluum Alternative"},
		SerpDorks:      []string{"site:t.me voluum"},
	}
	merged := MergeSuggestions(current, []string{"voluum alternative", "binom pain"}, []string{"site:t.me voluum", "site:t.me binom"})
	if len(merged.TelegramSearch) != 2 {
		t.Fatalf("telegram=%v", merged.TelegramSearch)
	}
	if len(merged.SerpDorks) != 2 {
		t.Fatalf("serp=%v", merged.SerpDorks)
	}
}
