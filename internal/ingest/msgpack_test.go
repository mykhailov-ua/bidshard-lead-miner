package ingest

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func TestParseMsgpackFrame(t *testing.T) {
	item := telegramItem{
		Source:    "telegram:@affnet",
		Text:      "voluum alternative",
		Username:  "buyer_mx",
		MessageID: 42,
		Contact:   "telegram:@buyer_mx",
	}
	raw, err := msgpack.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseMsgpackFrame(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != item.Source || got.MessageID != 42 {
		t.Fatalf("unexpected %+v", got)
	}
}

func TestReadMsgpackFrame(t *testing.T) {
	payload, err := msgpack.Marshal(telegramItem{Source: "telegram:@x", Text: "pain", MessageID: 1})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(payload)))
	buf.Write(payload)

	frame, err := readMsgpackFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	item, err := parseMsgpackFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if item.MessageID != 1 {
		t.Fatalf("message_id=%d", item.MessageID)
	}
}
