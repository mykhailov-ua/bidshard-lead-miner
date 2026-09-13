package filter

import "testing"

func TestRejectNonBuyerContext(t *testing.T) {
	if drop, _ := RejectNonBuyerContext("jobboard:jobs.dou.ua/tapok", "We are hiring media buyer", "Media Buyer"); drop {
		t.Fatal("expected jobboard source to bypass job-context drop")
	}
	drop, reason := RejectNonBuyerContext("forum:affiliatefix.com", "Hiring: media buyer for FB/Google team", "")
	if drop {
		t.Fatalf("allowlisted forum team hiring should pass: reason=%q", reason)
	}
	drop, reason = RejectNonBuyerContext("forum:random-blog.com", "We are hiring media buyer", "")
	if !drop || reason == "" {
		t.Fatalf("non-allowlist forum job post should drop: drop=%v reason=%q", drop, reason)
	}
	drop, _ = RejectNonBuyerContext("forum:affiliatefix", "Step by step tutorial how to build landing pages", "")
	if !drop {
		t.Fatal("tutorial without pain should drop")
	}
	drop, _ = RejectNonBuyerContext("forum:affiliatefix", "Step by step tutorial: alternative to voluum?", "")
	if drop {
		t.Fatal("tutorial with pain should pass")
	}
	drop, _ = RejectNonBuyerContext("github:org/repo", "bump version documentation update", "")
	if !drop {
		t.Fatal("github maintenance noise should drop")
	}
	drop, _ = RejectNonBuyerContext("github:org/repo", "postback not working on voluum migration", "")
	if drop {
		t.Fatal("github pain should pass")
	}
}

func TestTelegramChannelBroadcastReject(t *testing.T) {
	if TelegramChannelBroadcastReject("telegram:@news", "supergroup", 0, "daily casino tips") {
		t.Fatal("expected keep supergroup")
	}
	if !TelegramChannelBroadcastReject("telegram:@news", "channel", 0, "daily casino tips follow us") {
		t.Fatal("expected reject broadcast without dialog")
	}
	if TelegramChannelBroadcastReject("telegram:@news", "channel", 0, "alternative to voluum?") {
		t.Fatal("expected keep question with pain")
	}
	if TelegramChannelBroadcastReject("telegram:@news", "channel", 42, "random promo") {
		t.Fatal("expected keep reply thread")
	}
}
