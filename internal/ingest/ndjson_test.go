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

func TestParseTelegramSenderUserID(t *testing.T) {
	t.Parallel()

	line := []byte(`{"source":"telegram:@affnet","text":"voluum pain","message_id":3,"sender_user_id":99887766}`)
	item, err := parseNDJSONLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if got := item.ContactTelegram(); got != "telegram:user_id:99887766" {
		t.Fatalf("contact=%q", got)
	}
	if err := validateTelegramItem(item); err != nil {
		t.Fatal(err)
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

func TestParseTelegramReplyContext(t *testing.T) {
	t.Parallel()

	line := []byte(`{"source":"telegram:@affnet","text":"same here","username":"buyer","message_id":2,"reply_to_message_id":1,"reply_context":"parent voluum pain"}`)
	item, err := parseNDJSONLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if item.ReplyContext != "parent voluum pain" {
		t.Fatalf("reply_context=%q", item.ReplyContext)
	}
	if item.ReplyToMessageID != 1 {
		t.Fatalf("reply_to_message_id=%d", item.ReplyToMessageID)
	}
}
func TestParseTelegramChannelAbout(t *testing.T) {
	t.Parallel()

	line := []byte(`{"source":"telegram:@affnet","text":"voluum alternative","username":"buyer","message_id":1,"channel_about":"Global affiliate team EU"}`)
	item, err := parseNDJSONLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if item.ChannelAbout != "Global affiliate team EU" {
		t.Fatalf("channel_about=%q", item.ChannelAbout)
	}
}
