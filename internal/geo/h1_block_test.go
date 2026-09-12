package geo

import (
	"strings"
	"testing"
)

func TestFilterH1RejectBookmaker(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		text string
	}{
		{"1win promo", "Join our 1win funnel, best rates"},
		{"melbet", "Looking for melbet landing templates"},
		{"pin-up.ru", "Setup pin-up.ru funnels for RU geo"},
		{"vavada", "vavada creatives pack for sale"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := FilterH1(tc.text, "buyer_en", "")
			if res.OK {
				t.Fatal("expected cis_grey_market reject")
			}
			if !strings.HasPrefix(res.Reason, RejectCISGreyMarket) {
				t.Fatalf("reason=%q want prefix %q", res.Reason, RejectCISGreyMarket)
			}
		})
	}
}

func TestFilterH1BookmakerWWBuyerVoiceExempt(t *testing.T) {
	t.Parallel()
	text := "1win postbacks not matching voluum logs, USDT payout team needs proof"
	res := FilterH1(text, "buyer_ua", "")
	if !res.OK {
		t.Fatalf("expected pass with WW buyer voice, got %q", res.Reason)
	}
}

func TestFilterH1PassUATeam(t *testing.T) {
	t.Parallel()
	text := "Ищем media buyer в Киев, USDT TRC20, voluum postback не сходится"
	res := FilterH1(text, "team_lead", "")
	if !res.OK {
		t.Fatalf("expected UA team pass, got %q", res.Reason)
	}
}

func TestFilterH1RejectRuMetadata(t *testing.T) {
	t.Parallel()
	res := FilterH1("voluum alternative discussion", "affiliate_team.ru", "US team")
	if res.OK {
		t.Fatal("expected geo_block on .ru handle")
	}
	if !strings.HasPrefix(res.Reason, RejectGeoBlockRUBY) {
		t.Fatalf("reason=%q", res.Reason)
	}
}

func TestFilterH1RejectRuFlagInBio(t *testing.T) {
	t.Parallel()
	res := FilterH1("keitaro migration help", "buyer_en", "Media buyer 🇷🇺")
	if res.OK {
		t.Fatal("expected geo_block on ru flag")
	}
}

func TestFilterH1RejectRubCards(t *testing.T) {
	t.Parallel()
	res := FilterH1("оплата рублевые карты за трекер", "", "")
	if res.OK {
		t.Fatal("expected geo_block")
	}
}

func TestFilterH1CpaRipWithoutRUContextPasses(t *testing.T) {
	t.Parallel()
	res := FilterH1("discussion on cpa.rip forum about voluum pricing", "", "")
	if !res.OK {
		t.Fatalf("cpa.rip without ru context should pass, got %q", res.Reason)
	}
}

func TestFilterH1CpaRipRUSetupRejects(t *testing.T) {
	t.Parallel()
	res := FilterH1("cpa.rip RU setup for 1win funnels", "", "")
	if res.OK {
		t.Fatal("expected cis or geo reject")
	}
}

func TestFilterH1DoesNotMatchBare1x(t *testing.T) {
	t.Parallel()
	res := FilterH1("1x traffic multiplier on voluum postback failing", "buyer", "")
	if !res.OK {
		t.Fatalf("bare 1x should not trigger bookmaker, got %q", res.Reason)
	}
}
