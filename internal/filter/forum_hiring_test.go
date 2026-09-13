package filter

import "testing"

func TestForumTeamHiringBypass(t *testing.T) {
	if !ForumTeamHiringBypassContextDrop(
		"forum:affiliatefix.com/thread-1",
		"Hiring media buyer for paid social",
		"",
	) {
		t.Fatal("expected bypass on allowlisted host")
	}
	if ForumTeamHiringBypassContextDrop("forum:unknown.example", "Hiring media buyer", "") {
		t.Fatal("expected no bypass off allowlist")
	}
}
