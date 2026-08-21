package tgweb

import "testing"

func TestAggressivePrescanFromContactSkype(t *testing.T) {
	t.Parallel()
	if !AggressivePrescanFromContact("tgweb:@buylinkpro:buylink.pro", "skype:aff.manager") {
		t.Fatal("expected skype site lpr prescan pass")
	}
}

func TestAggressivePrescanFromContactSiteEmail(t *testing.T) {
	t.Parallel()
	if !AggressivePrescanFromContact("tgweb:@aff:topxpartners.com", "affiliates@topxpartners.com") {
		t.Fatal("expected on-domain email prescan pass")
	}
}

func TestAggressivePrescanRejectsOffDomainEmail(t *testing.T) {
	t.Parallel()
	if AggressivePrescanFromContact("tgweb:@aff:bojoko.com", "partners@other.net") {
		t.Fatal("expected off-domain email to fail")
	}
}

func TestAggressivePrescanRejectsNonTgweb(t *testing.T) {
	t.Parallel()
	if AggressivePrescanFromContact("lander:example.com", "a@example.com") {
		t.Fatal("expected non-tgweb to fail")
	}
}
