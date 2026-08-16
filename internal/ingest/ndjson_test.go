package ingest

import (
	"strings"
	"testing"
)

func TestParseTelegramNDJSON(t *testing.T) {
	t.Parallel()

	line := []byte(`{"source":"telegram:@affiliate_latam_en","text":"voluum alternative","username":"media_buyer_mx","message_id":1001}`)
	item, err := parseNDJSONLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if item.Source != "telegram:@affiliate_latam_en" {
		t.Fatalf("source=%q", item.Source)
	}
	if item.MessageID != 1001 {
		t.Fatalf("message_id=%d", item.MessageID)
	}
	if got := item.ContactTelegram(); got != "telegram:@media_buyer_mx" {
		t.Fatalf("contact=%q", got)
	}
}

func TestParseTelegramFromSourceOnly(t *testing.T) {
	t.Parallel()

	line := []byte(`{"source":"telegram:@buyer","text":"postback failing","message_id":9}`)
	item, err := parseNDJSONLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTelegramItem(item); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(item.ContactTelegram(), "telegram:@") {
		t.Fatalf("contact=%q", item.ContactTelegram())
	}
}

func TestValidateTelegramMissingContact(t *testing.T) {
	t.Parallel()

	item, _ := parseNDJSONLine([]byte(`{"source":"telegram:","text":"hello","message_id":1}`))
	if err := validateTelegramItem(item); err == nil {
		t.Fatal("expected validation error")
	}
}
