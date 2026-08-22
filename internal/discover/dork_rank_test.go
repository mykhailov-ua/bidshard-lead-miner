package discover

import "testing"

func TestMatchTelegramDork(t *testing.T) {
	idx := map[string]string{
		"affops": "site:t.me affops",
		"igchat": "telegram igaming",
	}
	if got := matchTelegramDork("forum:affiliatefix.com", idx); got != "" {
		t.Fatalf("non-telegram source: got %q", got)
	}
	if got := matchTelegramDork("telegram:affops/thread-1", idx); got != "site:t.me affops" {
		t.Fatalf("thread source: got %q", got)
	}
	if got := matchTelegramDork("telegram:@igchat", idx); got != "telegram igaming" {
		t.Fatalf("@ prefix: got %q", got)
	}
	if got := matchTelegramDork("telegram:unknown", idx); got != "" {
		t.Fatalf("unknown username: got %q", got)
	}
}
