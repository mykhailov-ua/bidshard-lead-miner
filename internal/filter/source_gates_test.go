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

func TestTelegramInviteWithoutBuyerIntent(t *testing.T) {
	t.Parallel()
	src := "telegram:invite:abc123"
	if !TelegramInviteWithoutBuyerIntent(src, "weekly news digest for affiliates") {
		t.Fatal("expected promo without intent to block")
	}
	if TelegramInviteWithoutBuyerIntent(src, "looking for voluum alternative for postback") {
		t.Fatal("expected buyer intent to pass")
	}
}
