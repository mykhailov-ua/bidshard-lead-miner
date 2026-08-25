package sourceregistry

import "testing"

func TestRelevanceScoreKeepBeatsDrop(t *testing.T) {
	t.Parallel()
	keep := RelevanceScore(Entry{TriageStatus: "keep", Types: []string{"tgweb", "supply"}})
	drop := RelevanceScore(Entry{TriageStatus: "drop"})
	if keep <= drop {
		t.Fatalf("keep=%d drop=%d", keep, drop)
	}
	if keep < 50 || keep > 100 {
		t.Fatalf("keep score=%d", keep)
	}
}

func TestSummarizeSkipsEmptyDomain(t *testing.T) {
	t.Parallel()
	rows := Summarize(File{Sources: []Entry{
		{Domain: "example.com", TriageStatus: "keep"},
		{Domain: "  "},
	}})
	if len(rows) != 1 || rows[0].Domain != "example.com" {
		t.Fatalf("rows=%+v", rows)
	}
}
