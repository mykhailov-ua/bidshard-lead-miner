package extract

import "testing"

func TestExtractForumUserHint(t *testing.T) {
	t.Parallel()

	got := Extract("voluum alternative postback failing", "forum:user/media_buyer")
	if len(got.Contacts) != 1 || got.Contacts[0].Type != "forum_user" {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}

func TestExtractRedditHint(t *testing.T) {
	t.Parallel()

	got := Extract("voluum alternative postback failing", "reddit:u/media_buyer")
	if len(got.Contacts) != 1 || got.Contacts[0].Type != "reddit" {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}

func TestExtractRejectsLinkedInOnly(t *testing.T) {
	t.Parallel()

	got := Extract("reach me https://linkedin.com/in/media-buyer voluum alternative")
	if !got.Rejected {
		t.Fatal("expected linkedin-only reject")
	}
	if got.Reason != "linkedin-only contact" {
		t.Fatalf("reason=%q", got.Reason)
	}
}

func TestExtractLinkedInMentionWithEmail(t *testing.T) {
	t.Parallel()

	got := Extract(
		"voluum alternative https://linkedin.com/in/media-buyer",
		"ops@igaming-team.com",
	)
	if got.Rejected {
		t.Fatalf("unexpected reject: %s", got.Reason)
	}
	if len(got.Contacts) != 1 || got.Contacts[0].Type != "email" {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}

func TestExtractEmailAndTelegram(t *testing.T) {
	t.Parallel()

	got := Extract("voluum alternative", "ops@igaming-team.com", "telegram:@buyer")
	if got.Rejected {
		t.Fatal("unexpected reject")
	}
	if len(got.Contacts) < 2 {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}

func TestExtractTMeLink(t *testing.T) {
	t.Parallel()

	got := Extract("voluum alternative contact t.me/buyer_mx")
	if got.Rejected {
		t.Fatal("unexpected reject")
	}
	found := false
	for _, c := range got.Contacts {
		if c.Type == "telegram" && c.Value == "@buyer_mx" {
			found = true
		}
	}
	if !found {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}

func TestExtractRejectsExampleEmail(t *testing.T) {
	t.Parallel()

	got := Extract("voluum alternative ops@example.com")
	if len(got.Contacts) != 0 {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}

func TestExtractDomainGitHubReviewHints(t *testing.T) {
	t.Parallel()

	got := Extract("voluum alternative tracker pain", "domain:click.example.com", "github:buyer42", "review:John D")
	if len(got.Contacts) != 3 {
		t.Fatalf("contacts=%v", got.Contacts)
	}
	types := map[string]string{}
	for _, c := range got.Contacts {
		types[c.Type] = c.Value
	}
	if types["domain"] != "click.example.com" {
		t.Fatalf("domain=%q", types["domain"])
	}
	if types["github"] != "buyer42" {
		t.Fatalf("github=%q", types["github"])
	}
	if types["review"] != "John D" {
		t.Fatalf("review=%q", types["review"])
	}
}

func TestExtractSkipsCSSTelegramHandles(t *testing.T) {
	t.Parallel()

	got := Extract("@media screen and (max-width: 768px) { } @keyframes fade { } @supports (display: grid)")
	for _, c := range got.Contacts {
		if c.Type == "telegram" {
			t.Fatalf("unexpected telegram contact %q", c.Value)
		}
	}
}

func TestExtractSkype(t *testing.T) {
	t.Parallel()
	got := Extract("voluum alternative contact Skype: media.buyer or live:aff_lead_mgr")
	if got.Rejected {
		t.Fatal("unexpected reject")
	}
	found := map[string]bool{}
	for _, c := range got.Contacts {
		if c.Type == "skype" {
			found[c.Value] = true
		}
	}
	if !found["media.buyer"] || !found["aff_lead_mgr"] {
		t.Fatalf("contacts=%v", got.Contacts)
	}
}
