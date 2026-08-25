package entity

import (
	"strings"
	"testing"
)

func TestBundleThreadMessagesSixFixture(t *testing.T) {
	t.Parallel()
	msgs := []string{
		"need voluum alternative",
		"postback broken on new offer",
		"looking for self-hosted tracker",
		"budget under 500/mo",
		"migration from keitaro",
		"who runs igaming funnels?",
	}
	got := BundleThreadMessages(msgs, DefaultTelegramThreadWindow)
	if got == "" {
		t.Fatal("expected bundled text")
	}
	for _, part := range msgs {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
}

func TestThreadBufferWindowSix(t *testing.T) {
	t.Parallel()
	buf := NewThreadBuffer(6)
	var last []string
	for i := 1; i <= 8; i++ {
		last = buf.Add("telegram:@aff", "buyer1", msgFor(i))
	}
	if len(last) != 6 {
		t.Fatalf("window len=%d", len(last))
	}
	if last[0] != msgFor(3) || last[5] != msgFor(8) {
		t.Fatalf("window=%v", last)
	}
	bundled := BundleThreadMessages(last, 0)
	if !strings.Contains(bundled, msgFor(8)) || !strings.Contains(bundled, msgFor(3)) {
		t.Fatalf("bundled=%q", bundled)
	}
}

func msgFor(i int) string {
	return []string{
		"need voluum alternative",
		"postback broken on new offer",
		"looking for self-hosted tracker",
		"budget under 500/mo",
		"migration from keitaro",
		"who runs igaming funnels?",
		"binom vs voluum?",
		"need EU geo targeting",
	}[i-1]
}
