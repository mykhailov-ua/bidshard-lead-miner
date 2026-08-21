package store

import "testing"

func TestNormalizeTag(t *testing.T) {
	tag, err := normalizeTag("High-Priority")
	if err != nil || tag != "high-priority" {
		t.Fatalf("tag=%q err=%v", tag, err)
	}
	if _, err := normalizeTag("bad tag"); err == nil {
		t.Fatal("expected error for space in tag")
	}
}

func TestNormalizeNoteText(t *testing.T) {
	text, err := normalizeNoteText("  hello   world  ")
	if err != nil || text != "hello world" {
		t.Fatalf("text=%q err=%v", text, err)
	}
	if _, err := normalizeNoteText("   "); err == nil {
		t.Fatal("expected error for empty note")
	}
}
