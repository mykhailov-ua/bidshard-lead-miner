package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func benchQueries() ([]string, []string) {
	base := make([]string, 0, 32)
	extra := make([]string, 0, 32)
	for i := 0; i < 16; i++ {
		base = append(base, "Voluum Alternative "+string(rune('A'+i)))
		extra = append(extra, "voluum alternative "+string(rune('a'+i)))
	}
	return base, extra
}

func BenchmarkMergeSuggestions(b *testing.B) {
	base, _ := benchQueries()
	current := ICPConfig{TelegramSearch: base, SerpDorks: base}
	tg, serp := benchQueries()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = MergeSuggestions(current, tg, serp)
	}
}

func BenchmarkWritePending(b *testing.B) {
	dir := b.TempDir()
	diff := PendingICPDiff{
		AddTelegramSearch: []string{"binom migration", "keitaro pain"},
		AddSerpDorks:      []string{"site:t.me binom"},
		Summary:           "expand queries",
		ReportID:          "bench-report",
		GeneratedAt:       "2026-01-01T00:00:00Z",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sub := filepath.Join(dir, "run")
		_ = os.MkdirAll(sub, 0o755)
		_, _ = WritePending(sub, "bench-report", diff)
	}
}
