package validate

import "testing"

func TestAcceptEmailRejectsJunk(t *testing.T) {
	t.Parallel()

	rejects := []string{
		"banner@cdn.example.png",
		"ops@example.com",
		"bad-email",
		"noreply@users.noreply.github.com",
		"support@igaming-team.com",
		"sales@acme.com",
		"marketing@acme.com",
		"accounting@cpa-firm.com",
		"info@acme.com",
	}
	for _, email := range rejects {
		if AcceptEmail(email) {
			t.Fatalf("expected reject for %q", email)
		}
	}
	accepts := []string{
		"ops@igaming-team.com",
		"ads@acme.com",
		"partnerships@buylink.pro",
		"affiliates@mail.bojoko.com",
		"media@network.io",
	}
	for _, email := range accepts {
		if !AcceptEmail(email) {
			t.Fatalf("expected accept for %q", email)
		}
	}
	if !AcceptEmail("ceo@igaming-team.com") {
		t.Fatal("expected executive email")
	}
	if AcceptEmail("buyer+test@gmail.com") {
		t.Fatal("expected reject for gmail plus tag")
	}
}

func TestIsRoleEmail(t *testing.T) {
	t.Parallel()
	affiliateB2B := []string{
		"ads@acme.com",
		"partnerships@buylink.pro",
		"affiliates@bojoko.com",
		"bizdev@network.io",
	}
	for _, email := range affiliateB2B {
		if IsRoleEmail(email) {
			t.Fatalf("%q should not be generic role", email)
		}
	}
	if !IsRoleEmail("sales.manager@acme.com") {
		t.Fatal("sales* prefix should be role")
	}
	if !IsRoleEmail("info@acme.com") {
		t.Fatal("info@ should be role")
	}
	if IsRoleEmail("buyer@acme.com") {
		t.Fatal("buyer@ should not be role")
	}
	if IsRoleEmail("ceo@acme.com") {
		t.Fatal("ceo@ should not be role")
	}
}
