package sources

import (
	"context"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type StubSource struct {
	name     string
	items    []model.RawItem
	delay    time.Duration
	blocking bool
}

func NewStubSource(name string, items []model.RawItem) *StubSource {
	return &StubSource{name: name, items: items}
}

func NewBlockingStubSource(name string) *StubSource {
	return &StubSource{name: name, blocking: true}
}

func (s *StubSource) Name() string {
	return s.name
}

func (s *StubSource) Collect(ctx context.Context, emit EmitFunc) error {
	if s.blocking {
		<-ctx.Done()
		return ctx.Err()
	}

	if s.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.delay):
		}
	}

	for _, item := range s.items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		item.Source = s.name
		if err := emit(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func DefaultStubs() []Source {
	return []Source{
		NewStubSource("stub:telegram_en", []model.RawItem{
			{Raw: "voluum alternative needed. postback failing on FTD again", Contact: "telegram:@buyer"},
			{Raw: "Looking for self-hosted tracker. voluum alternative", Contact: "telegram:@media_buyer_mx"},
		}),
		NewStubSource("stub:ads_txt", []model.RawItem{
			{Raw: "Voluum pricing too high. postback failing every night", Contact: "ops@igaming-team.com"},
		}),
	}
}
