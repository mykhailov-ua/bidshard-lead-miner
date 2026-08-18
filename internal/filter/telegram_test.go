package filter

import "testing"

func TestTelegramSpamShortWithoutPain(t *testing.T) {
	t.Parallel()
	if spam, _ := TelegramSpam("telegram:@chat", "hi team"); !spam {
		t.Fatal("expected spam")
	}
}

func TestTelegramSpamAllowsPainfulShort(t *testing.T) {
	t.Parallel()
	if spam, _ := TelegramSpam("telegram:@chat", "voluum alternative postback failing"); spam {
		t.Fatal("expected pass")
	}
}

func TestTelegramSpamChannelPromo(t *testing.T) {
	t.Parallel()
	if spam, reason := TelegramSpam("telegram:@chat", "join our channel for free signals"); !spam || reason == "" {
		t.Fatalf("spam=%v reason=%q", spam, reason)
	}
}

func TestNonTelegramSkipped(t *testing.T) {
	t.Parallel()
	if spam, _ := TelegramSpam("forum:stm", "join our channel"); spam {
		t.Fatal("expected pass for non-telegram")
	}
}
