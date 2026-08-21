package admin

import "testing"

func TestParseInboxOnly(t *testing.T) {
	t.Parallel()

	if !parseInboxOnly("", "new") {
		t.Fatal("expected inbox default on for status=new")
	}
	if parseInboxOnly("false", "new") {
		t.Fatal("expected inbox off when inbox=false")
	}
	if !parseInboxOnly("true", "") {
		t.Fatal("expected inbox on when inbox=true")
	}
	if parseInboxOnly("", "contacted") {
		t.Fatal("expected inbox off for non-new status without explicit inbox")
	}
}
