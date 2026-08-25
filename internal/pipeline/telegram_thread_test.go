package pipeline

import (
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/model"
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

func TestTelegramThreadTextNonTelegramPassthrough(t *testing.T) {
	t.Parallel()
	p := &Processor{TelegramThread: entity.NewThreadBuffer(6)}
	task := Task{Item: model.RawItem{Source: "reddit:igaming", Raw: "plain"}}
	if got := p.telegramThreadText(task, "plain"); got != "plain" {
		t.Fatalf("got=%q", got)
	}
}
