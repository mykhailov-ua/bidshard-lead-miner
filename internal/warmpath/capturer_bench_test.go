package warmpath

import (
	"strings"
	"testing"
)

func BenchmarkTryCapture(b *testing.B) {
	c := NewCapturer(4096)
	ev := Event{
		HashID:  "bench-hash",
		Source:  "telegram:bench",
		Snippet: strings.Repeat("voluum alternative postback pain ", 40),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.TryCapture(ev)
	}
}
