package extract

import "testing"

func TestTelegramHandles(t *testing.T) {
	t.Parallel()
	text := "Join https://t.me/affiliate_latam and @media_buyer_mx or telegram.me/igaming_acquisition"
	got := TelegramHandles(text)
	want := map[string]bool{
		"affiliate_latam":     true,
		"media_buyer_mx":      true,
		"igaming_acquisition": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %d handles", got, len(want))
	}
	for _, u := range got {
		if !want[u] {
			t.Fatalf("unexpected handle %q in %v", u, got)
		}
	}
}

func TestTelegramHandlesSkipsBots(t *testing.T) {
	t.Parallel()
	got := TelegramHandles("contact @helper_bot or t.me/RealChannel")
	if len(got) != 1 || got[0] != "realchannel" {
		t.Fatalf("got %v", got)
	}
}

func TestTelegramInviteHashes(t *testing.T) {
	t.Parallel()
	text := "join https://t.me/+AbCdEfGhIjKlMn and https://t.me/joinchat/OldStyleHash12"
	got := TelegramInviteHashes(text)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}
