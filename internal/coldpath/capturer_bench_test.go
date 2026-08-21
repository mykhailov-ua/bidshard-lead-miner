package coldpath

import (
	"strings"
	"testing"
)

func BenchmarkTryCapture(b *testing.B) {
	c := NewCapturer(4096)
	ev := Event{
		Source:      "telegram:bench",
		Reason:      "low_score",
		Snippet:     strings.Repeat("voluum alternative postback pain ", 40),
		ContactHint: "buyer@example.com",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.TryCapture(ev)
	}
}
