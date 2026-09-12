package filter

import "testing"

func TestTelegramReplyThreadBuyer(t *testing.T) {
	t.Parallel()
	parent := "need voluum alternative, postback failing"
	if !TelegramReplyThreadBuyer(12, parent, "same issue here", "buyer_mx") {
		t.Fatal("expected buyer reply in pain thread")
	}
	if TelegramReplyThreadBuyer(12, parent, "same issue", "") {
		t.Fatal("expected reject without username")
	}
	if TelegramReplyThreadBuyer(0, parent, "same issue", "buyer_mx") {
		t.Fatal("expected reject without reply_to")
	}
	if TelegramReplyThreadBuyer(12, "daily news digest", "same issue", "buyer_mx") {
		t.Fatal("expected reject without parent pain")
	}
}

func TestTelegramReplyHelperReject(t *testing.T) {
	t.Parallel()
	drop, reason := TelegramReplyHelperReject("DM me for free course on tracker setup")
	if !drop || reason == "" {
		t.Fatalf("drop=%v reason=%q", drop, reason)
	}
	if drop, _ := TelegramReplyHelperReject("same voluum pain here"); drop {
		t.Fatal("expected buyer reply keep")
	}
}
