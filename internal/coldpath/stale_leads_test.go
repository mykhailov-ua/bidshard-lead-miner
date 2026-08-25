package coldpath

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
)

type stubStaleLeadLister struct {
	docs []sink.LeadDoc
}

func (s *stubStaleLeadLister) ListStaleNewLeads(context.Context, time.Duration, int) ([]sink.LeadDoc, error) {
	return s.docs, nil
}

type stubStaleLeadPatcher struct {
	patches []sink.LeadAnalysisPatch
}

func (s *stubStaleLeadPatcher) PatchLeadAnalysis(_ context.Context, patch sink.LeadAnalysisPatch) error {
	s.patches = append(s.patches, patch)
	return nil
}

type stubICPClassifier struct {
	res gemini.ICPResult
}

func (s *stubICPClassifier) ClassifyICP(context.Context, string) (gemini.ICPResult, error) {
	return s.res, nil
}

func TestStaleLeadRegraderMarksStaleWhenICPNone(t *testing.T) {
	patcher := &stubStaleLeadPatcher{}
	r := NewStaleLeadRegrader(time.Hour, 30*24*time.Hour, 10,
		&stubStaleLeadLister{docs: []sink.LeadDoc{{
			HashID:  "hash-old",
			Snippet: "old affiliate thread with no buyer pain",
			Score:   10,
		}}},
		patcher,
		&stubICPClassifier{res: gemini.ICPResult{ICP: "none", Hot: false, Why: "no fit"}},
		scoring.NewRegistry("../../testdata/keywords.json"),
	)
	r.RunStaleRegradeOnce(t.Context())
	if len(patcher.patches) != 1 {
		t.Fatalf("patches=%d want 1", len(patcher.patches))
	}
	if !patcher.patches[0].Stale {
		t.Fatal("expected stale=true when ICP none and low priority")
	}
}
