package extract

import "testing"

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
