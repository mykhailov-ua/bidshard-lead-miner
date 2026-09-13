package osint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergePeopleFiles_profileAndMembers(t *testing.T) {
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles.json")
	members := filepath.Join(dir, "members.json")

	if err := os.WriteFile(profiles, []byte(`{
  "profiles": [{
    "user_id": 42,
    "username": "buyer_jane",
    "cold_outreach_fit": "high",
    "discovered_via": "profile_link:bio",
    "source_chat": "mediabuyers"
  }]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(members, []byte(`{
  "members": [{
    "user_id": 42,
    "username": "buyer_jane",
    "chat_username": "cpa_chat",
    "chat_key": "u:cpa_chat"
  }, {
    "user_id": 99,
    "username": "solo_guy",
    "chat_username": "offers_ru"
  }]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	rows, err := MergePeopleFiles(profiles, members)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2", len(rows))
	}
	byID := map[int64]struct {
		via, fit, source, member, sourceRef, memberKey string
	}{}
	for _, r := range rows {
		byID[r.UserID] = struct {
			via, fit, source, member, sourceRef, memberKey string
		}{r.DiscoveredVia, r.OutreachFit, r.SourceChat, r.MemberChat, r.SourceChatRef, r.MemberChatKey}
	}
	p42 := byID[42]
	if p42.via != "profile_link:bio" {
		t.Errorf("user 42 discovered_via=%q", p42.via)
	}
	if p42.fit != "high" {
		t.Errorf("user 42 outreach_fit=%q", p42.fit)
	}
	if p42.source != "mediabuyers" {
		t.Errorf("user 42 source_chat=%q", p42.source)
	}
	if p42.member != "cpa_chat" {
		t.Errorf("user 42 member_chat=%q", p42.member)
	}
	if p42.sourceRef != "tg_chat:u:mediabuyers" {
		t.Errorf("user 42 source_chat_ref=%q", p42.sourceRef)
	}
	if p42.memberKey != "u:cpa_chat" {
		t.Errorf("user 42 member_chat_key=%q", p42.memberKey)
	}
	p99 := byID[99]
	if p99.via != "member_harvest" {
		t.Errorf("user 99 discovered_via=%q want member_harvest", p99.via)
	}
}
