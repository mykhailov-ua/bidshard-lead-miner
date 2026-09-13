package store

import "testing"

func TestTelegramPersonKey(t *testing.T) {
	if TelegramPersonKey(0) != "" {
		t.Fatal("zero user id")
	}
	if TelegramPersonKey(42) != "tg_user:42" {
		t.Fatalf("got %q", TelegramPersonKey(42))
	}
}

func TestTelegramChatRef(t *testing.T) {
	if TelegramChatRef("u:aff_chat") != "tg_chat:u:aff_chat" {
		t.Fatal(TelegramChatRef("u:aff_chat"))
	}
	if TelegramChatRefFromUsername("@Aff_Chat") != "tg_chat:u:aff_chat" {
		t.Fatal(TelegramChatRefFromUsername("@Aff_Chat"))
	}
}

func TestTelegramMemberEdgeKey(t *testing.T) {
	edge := TelegramMemberEdgeKey("tg_chat:u:cpa", "tg_user:99")
	if edge != "tg_member:tg_chat:u:cpa|tg_user:99" {
		t.Fatalf("edge=%q", edge)
	}
}

func TestFillTelegramPersonCanonicalKeys_legacyRow(t *testing.T) {
	doc := &TelegramPersonDoc{
		UserID:      7,
		SourceChat:  "mediabuyers",
		MemberChats: []string{"cpa_chat", "mediabuyers"},
	}
	FillTelegramPersonCanonicalKeys(doc)
	if doc.PersonKey != "tg_user:7" {
		t.Fatalf("person_key=%q", doc.PersonKey)
	}
	if doc.SourceChatRef != "tg_chat:u:mediabuyers" {
		t.Fatalf("source_chat_ref=%q", doc.SourceChatRef)
	}
	if len(doc.MemberChatRefs) != 2 {
		t.Fatalf("member_chat_refs=%v", doc.MemberChatRefs)
	}
	if len(doc.MemberEdges) != 2 {
		t.Fatalf("member_edges=%v", doc.MemberEdges)
	}
}
