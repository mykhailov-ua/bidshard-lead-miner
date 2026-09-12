package model

import "testing"

func TestContactTelegramIgnoresSerpPlaceholder(t *testing.T) {
	t.Parallel()
	item := RawItem{Contact: "serp:www.redtrack.io"}
	if item.ContactTelegram() != "" {
		t.Fatalf("expected empty telegram contact, got %q", item.ContactTelegram())
	}
}
