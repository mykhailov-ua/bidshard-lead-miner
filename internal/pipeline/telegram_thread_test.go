package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

func TestTelegramThreadTextBundlesSixMessages(t *testing.T) {
	t.Parallel()
	p := &Processor{TelegramThread: entity.NewThreadBuffer(6)}
	task := Task{Item: model.RawItem{
		Source:  "telegram:@affnet",
		Contact: "telegram:@buyer_mx",
	}}
	var last string
	for i := 1; i <= 6; i++ {
		last = p.telegramThreadText(task, msgForThread(i))
	}
	if last == "" {
		t.Fatal("expected bundled text")
	}
	for i := 1; i <= 6; i++ {
		if !strings.Contains(last, msgForThread(i)) {
			t.Fatalf("missing msg %d in %q", i, last)
		}
	}
}

func msgForThread(i int) string {
	return []string{
		"need voluum alternative",
		"postback broken on new offer",
		"looking for self-hosted tracker",
		"budget under 500/mo",
		"migration from keitaro",
		"who runs igaming funnels?",
	}[i-1]
}

func TestTelegramThreadTextIncludesReplyContext(t *testing.T) {
	t.Parallel()
	p := &Processor{TelegramThread: entity.NewThreadBuffer(6)}
	task := Task{Item: model.RawItem{
		Source:       "telegram:@affnet",
		Contact:      "telegram:@buyer_mx",
		ReplyContext: "parent needs voluum alternative",
	}}
	got := p.telegramThreadText(task, "same tracker pain here")
	if !strings.Contains(got, "reply_to: parent needs voluum alternative") {
		t.Fatalf("missing reply context in %q", got)
	}
	if !strings.Contains(got, "same tracker pain here") {
		t.Fatalf("missing message text in %q", got)
	}
}

func TestProcessorAcceptsTelegramReplyThreadBuyer(t *testing.T) {
	t.Parallel()
	reg := loadTestRegistry(t)
	store := sink.NewStubStore()
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:           "telegram:@affiliate_igaming",
			Raw:              "same issue here, anyone else?",
			Username:         "buyer_mx",
			Contact:          "telegram:@buyer_mx",
			ReplyToMessageID: 10,
			ReplyContext:     "need voluum alternative, postback failing on FTD",
			ChatType:         "supergroup",
		},
	})
	if !out.Accepted {
		t.Fatalf("expected accept, reason=%q", out.RejectReason)
	}
}

func TestProcessorRejectsTelegramReplyHelper(t *testing.T) {
	t.Parallel()
	reg := loadTestRegistry(t)
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:           "telegram:@affiliate_igaming",
			Raw:              "DM me for free course on tracker setup",
			Username:         "helper_svc",
			Contact:          "telegram:@helper_svc",
			ReplyToMessageID: 10,
			ReplyContext:     "need voluum alternative, postback failing",
			ChatType:         "supergroup",
		},
	})
	if out.Accepted {
		t.Fatal("expected helper reply reject")
	}
}

func TestTelegramThreadTextNonTelegramPassthrough(t *testing.T) {
	t.Parallel()
	p := &Processor{TelegramThread: entity.NewThreadBuffer(6)}
	task := Task{Item: model.RawItem{Source: "reddit:igaming", Raw: "plain"}}
	if got := p.telegramThreadText(task, "plain"); got != "plain" {
		t.Fatalf("got=%q", got)
	}
}
