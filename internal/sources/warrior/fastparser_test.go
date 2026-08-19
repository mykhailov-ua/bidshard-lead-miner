package warrior

import (
	"regexp"
	"strings"
	"testing"
)

var benchmarkHTML = []byte(`
<div class="post-content">
	<p>Hello, we are looking for a reliable <strong>Voluum alternative</strong> with low postback latency!</p>
	<a href="https://example.com/ref" class="username">AffiliatePro</a>
	<span>Contact us via Telegram @aff_lead_manager or Skype.</span>
</div>
`)

func LegacyStripTags(s string) string {
	s = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func TestFastHTMLStripper(t *testing.T) {
	stripper := NewFastHTMLStripper(4096)
	got := stripper.StripTagsBytes(benchmarkHTML)
	if !strings.Contains(got, "Voluum alternative") {
		t.Errorf("expected Voluum alternative in stripped output, got: %s", got)
	}
	if strings.Contains(got, "<p>") || strings.Contains(got, "</div>") {
		t.Errorf("found unstripped tags in output: %s", got)
	}
}

func BenchmarkLegacyStripTags(b *testing.B) {
	input := string(benchmarkHTML)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LegacyStripTags(input)
	}
}

func BenchmarkFastHTMLStripper(b *testing.B) {
	stripper := NewFastHTMLStripper(4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stripper.StripTagsBytes(benchmarkHTML)
	}
}
