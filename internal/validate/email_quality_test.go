package validate

import "testing"

func TestAcceptEmailRejectsJunk(t *testing.T) {
	t.Parallel()

	rejects := []string{
		"banner@cdn.example.png",
		"ops@example.com",
		"bad-email",
		"noreply@users.noreply.github.com",
	}
	for _, email := range rejects {
		if AcceptEmail(email) {
			t.Fatalf("expected reject for %q", email)
		}
	}
	if !AcceptEmail("ops@igaming-team.com") {
		t.Fatal("expected valid email")
	}
	if AcceptEmail("buyer+test@gmail.com") {
		t.Fatal("expected reject for gmail plus tag")
	}
}

func TestIsRoleEmail(t *testing.T) {
	t.Parallel()
	if !IsRoleEmail("ads@acme.com") {
		t.Fatal("ads@ should be role")
	}
	if IsRoleEmail("buyer@acme.com") {
		t.Fatal("buyer@ should not be role")
	}
}
