package warmpath

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/coldpath"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

type stubWarmPrescan struct {
	spam bool
}

func (s stubWarmPrescan) EvaluateSpam(context.Context, string) (gemini.PrescanVerdict, error) {
	return gemini.PrescanVerdict{SpamMatch: s.spam, SpamScore: 0.95}, nil
}

type recordingWarmJunk struct {
	count int
}

func (r *recordingWarmJunk) InsertMany(_ context.Context, docs []sink.JunkDoc) error {
	r.count += len(docs)
	return nil
}

func TestFilterWarmPrescanRejectsSpam(t *testing.T) {
	t.Parallel()

	patcher := &recordingPatcher{}
	junk := &recordingWarmJunk{}
	svc := &Service{
		prescan: stubWarmPrescan{spam: true},
		patcher: patcher,
		junk:    junk,
	}
	batch := []Event{{HashID: "h1", Priority: "Medium", Snippet: "join vip signals"}}
	out := svc.filterWarmPrescan(context.Background(), batch)
	if len(out) != 0 {
		t.Fatalf("expected spam filtered, got %d", len(out))
	}
	if junk.count != 1 {
		t.Fatalf("junk inserts=%d want 1", junk.count)
	}
	if patch := patcher.last(); patch.AnalysisStatus != coldpath.ReasonWarmPrescanSpam {
		t.Fatalf("status=%q", patch.AnalysisStatus)
	}
}

func TestFilterWarmPrescanPassesClean(t *testing.T) {
	t.Parallel()

	svc := &Service{prescan: stubWarmPrescan{spam: false}}
	batch := []Event{{HashID: "h2", Priority: "High", Snippet: "voluum alternative"}}
	out := svc.filterWarmPrescan(context.Background(), batch)
	if len(out) != 1 {
		t.Fatalf("got %d events", len(out))
	}
}
