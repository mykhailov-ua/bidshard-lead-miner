package telegrambot

import "testing"

func TestSplitTelegramHTML(t *testing.T) {
	t.Parallel()
	short := "hello"
	if parts := SplitTelegramHTML(short, 4096); len(parts) != 1 || parts[0] != short {
		t.Fatalf("got %v", parts)
	}
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	parts := SplitTelegramHTML(string(long), 2000)
	if len(parts) < 3 {
		t.Fatalf("expected multiple chunks, got %d", len(parts))
	}
	for _, p := range parts {
		if len(p) > 2000 {
			t.Fatalf("chunk too long: %d", len(p))
		}
	}
}
