package pretty

import "testing"

func BenchmarkTruncate(b *testing.B) {
	s := "voluum alternative postback pain " + string(make([]byte, 2048))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Truncate(s, 500)
	}
}
