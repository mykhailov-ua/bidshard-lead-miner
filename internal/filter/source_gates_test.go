package filter

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/validate"
)

func TestTelegramChannelSelfBroadcast(t *testing.T) {
	t.Parallel()
	contacts := []extract.Contact{{Type: "telegram", Value: "@voluum"}}
	if !TelegramChannelSelfBroadcast("telegram:@voluum", contacts) {
		t.Fatal("expected channel self broadcast")
	}
	contacts = []extract.Contact{
		{Type: "telegram", Value: "@voluum"},
		{Type: "telegram", Value: "@buyer_mx"},
	}
	if TelegramChannelSelfBroadcast("telegram:@voluum", contacts) {
		t.Fatal("expected mention of other user to pass")
	}
}

func TestLanderRequiresEmailOrSkype(t *testing.T) {
	t.Parallel()
	if LanderRequiresEmailOrSkype([]extract.Contact{{Type: "telegram", Value: "@buyer"}}) {
		t.Fatal("telegram-only lander should fail")
	}
	if !LanderRequiresEmailOrSkype([]extract.Contact{{Type: "email", Value: "ops@acme.com"}}) {
		t.Fatal("email should pass")
	}
}

func TestGitHubRequiresPainContext(t *testing.T) {
	t.Parallel()
	if GitHubRequiresPainContext("github:ckan/ckan", "docker install permission error") {
		t.Fatal("expected CKAN issue without pain to fail")
	}
	if GitHubRequiresPainContext("github:stopthessotax/sso-wall-of-shame", "self-hosted SSO tax tracker pricing") {
		t.Fatal("expected SSO infra issue to fail")
	}
	if GitHubRequiresPainContext("github:keitaroinc/docker-ckan", "How is the envvars mechanism supposed to work?") {
		t.Fatal("expected vendor org issue to fail")
	}
	if !GitHubRequiresPainContext("github:ckan/ckan", "voluum postback failing on self-hosted tracker") {
		t.Fatal("expected pain context to pass")
	}
}

func TestGitHubVendorOrgBlocksNamespace(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source  string
		blocked bool
	}{
		{"github:keitaroinc/docker-ckan", true},
		{"github:keitaroinc/ckan", true},
		{"github:voluum/voluum-api", true},
		{"github:ckan/ckan", false},
		{"github:someuser/keitaro-alternative", false},
	}
	for _, tc := range cases {
		if got := GitHubVendorOrg(tc.source); got != tc.blocked {
			t.Errorf("GitHubVendorOrg(%q)=%v want %v", tc.source, got, tc.blocked)
		}
	}
}

func TestLanderRequiresBuyerSignal(t *testing.T) {
	t.Parallel()
	marketing := "voluum media buyer igaming affiliate s2s postback cost sync pricing"
	if LanderRequiresBuyerSignal(marketing) {
		t.Fatal("expected marketing copy without buyer signal to fail")
	}
	buyer := "We are looking for a voluum alternative because postback is failing"
	if !LanderRequiresBuyerSignal(buyer) {
		t.Fatal("expected buyer signal to pass")
	}
}

func TestLanderBlacklistedSource(t *testing.T) {
	t.Parallel()
	if err := validate.LoadBlacklistDomains("../../data/blacklist_domains.txt"); err != nil {
		t.Fatalf("load blacklist: %v", err)
	}
	if !LanderBlacklistedSource("lander:voluum.com") {
		t.Fatal("expected voluum.com lander blocked")
	}
	if LanderBlacklistedSource("lander:buyer-site.com") {
		t.Fatal("expected non-blacklisted lander to pass")
	}
}

func TestTelegramChannelSelfBroadcastUserIDReply(t *testing.T) {
	t.Parallel()
	contacts := []extract.Contact{
		{Type: "telegram", Value: "user_id:123456789"},
	}
	if TelegramChannelSelfBroadcast("telegram:@voluum", contacts) {
		t.Fatal("user_id reply must not count as self-broadcast")
	}
}

func TestTelegramIntelOnlyChannel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source string
		want   bool
	}{
		{"telegram:@igaming_news", true},
		{"telegram:@partnerkin_job", true},
		{"telegram:@affiliatechannel_igaming", true},
		{"telegram:@partneroff_pro", true},
		{"telegram:@voluum", false},
		{"telegram:invite:abc", false},
		{"reddit:r/test", false},
	}
	for _, tc := range cases {
		if got := TelegramIntelOnlyChannel(tc.source); got != tc.want {
			t.Errorf("TelegramIntelOnlyChannel(%q)=%v want %v", tc.source, got, tc.want)
		}
	}
}

func TestTelegramAgencyOutreachReject(t *testing.T) {
	t.Parallel()
	src := "telegram:@voluum"
	soak := "Hey I help eCommerce, Brand, enterprise & affiliate businesses grow through effective Ads campaign Google, Meta, Tiktok, Bing, LinkedIn Ads. Cloaking RedTrack, Voluum, Binom"
	if !TelegramAgencyOutreachReject(src, soak) {
		t.Fatal("expected agency promo soak snippet to block")
	}
	if TelegramAgencyOutreachReject(src, "Is there an option to resend a postback") {
		t.Fatal("expected buyer question to pass")
	}
	if TelegramAgencyOutreachReject(src, "looking for voluum alternative for postback") {
		t.Fatal("expected buyer intent to pass")
	}
	if TelegramAgencyOutreachReject("reddit:r/test", soak) {
		t.Fatal("expected non-telegram source to pass")
	}
}

func TestTelegramInviteWithoutBuyerIntent(t *testing.T) {
	t.Parallel()
	src := "telegram:invite:abc123"
	if !TelegramInviteWithoutBuyerIntent(src, "weekly news digest for affiliates") {
		t.Fatal("expected promo without intent to block")
	}
	if TelegramInviteWithoutBuyerIntent(src, "looking for voluum alternative for postback") {
		t.Fatal("expected buyer intent to pass")
	}
	if !TelegramInviteWithoutBuyerIntent(src, "Шукаємо Media Buyer у Gambling Sources: FB Ads budget $50k+/міс") {
		t.Fatal("expected vacancy spam without tracker pain to block")
	}
	if TelegramInviteWithoutBuyerIntent(src, "Шукаємо Tech спеціаліста для Keitaro та postback інтеграцій") {
		t.Fatal("expected vacancy with tracker pain to pass")
	}
	if !TelegramInviteWithoutBuyerIntent(src, "Мануал: как работать с трекером Keitaro. Подготовили для вас мануал по работе с трекером.") {
		t.Fatal("expected trafftok-style keitaro manual to block")
	}
	if !TelegramInviteWithoutBuyerIntent(src, "Мастхев скіли в Claude для кожного арбітражника-вайбкодера. claude keitaro skills.") {
		t.Fatal("expected vibe-coding tutorial invite to block")
	}
}
