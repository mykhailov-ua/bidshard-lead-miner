package filter

import "testing"

func TestInstantDropSellerSpam(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want bool
	}{
		{"Selling warmed BMs and agency accounts, DM me", true},
		{"Креативы на заказ, монтаж reels, озвучка видео", true},
		{"Набор в тиму, обучение с нуля, процент с профита", true},
		{"Keitaro postback timeout after traffic spike on nginx", false},
		{"Looking for voluum alternative, postback failing", false},
	}
	for _, tc := range cases {
		got, _ := InstantDropSellerSpam(tc.text)
		if got != tc.want {
			t.Fatalf("InstantDropSellerSpam(%q)=%v want %v", tc.text, got, tc.want)
		}
	}
}

func TestSellerAuthorProfile(t *testing.T) {
	t.Parallel()
	if drop, _ := SellerAuthorProfile("fb_accounts_shop", "", ""); !drop {
		t.Fatal("expected shop username drop")
	}
	if drop, _ := SellerAuthorProfile("media_buyer_ops", "nginx tuning", ""); drop {
		t.Fatal("expected buyer username pass")
	}
	if drop, _ := SellerAuthorProfile("", "Account Store - rent BM", ""); !drop {
		t.Fatal("expected store bio drop")
	}
}

func TestHasBanContextSignal(t *testing.T) {
	t.Parallel()
	if !HasBanContextSignal("FB ban wave again, pixel not seeing deps on CAPI") {
		t.Fatal("expected ban context")
	}
	if HasBanContextSignal("virtual cards for sale") {
		t.Fatal("seller spam is not ban context")
	}
}

func TestInfraPainBypassPrescan(t *testing.T) {
	t.Parallel()
	if !InfraPainBypassPrescan("502 Bad Gateway on tracker redirect, upstream timed out") {
		t.Fatal("expected infra bypass")
	}
}

func TestTeamBuyingSignal(t *testing.T) {
	t.Parallel()
	if !TeamBuyingSignal("Our media buying team runs Keitaro on dedicated VPS", "") {
		t.Fatal("expected team + tracker signal")
	}
	if TeamBuyingSignal("Selling warmed BMs, DM for price", "") {
		t.Fatal("seller spam should not qualify as team")
	}
	if !TeamBuyingSignal("Head of acquisition looking for voluum alternative", "") {
		t.Fatal("expected role + pain signal")
	}
}

func TestTechnicalAuthorSignal(t *testing.T) {
	t.Parallel()
	short := "nginx error?"
	if TechnicalAuthorSignal(short, "") {
		t.Fatal("short text should not qualify")
	}
	long := "We run keitaro behind nginx and started getting 502 Bad Gateway under load. " +
		"worker_connections is 4096, htop shows clickhouse cpu 100%. Here is our server block config - what am I missing?"
	if !TechnicalAuthorSignal(long, "nginx upstream timeout") {
		t.Fatal("expected technical author signal")
	}
}
