package scoring

import (
	"context"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	r := NewRegistry("../../testdata/keywords.json")
	if err := r.Load(context.Background()); err != nil {
		panic(err)
	}
	SetRegistry(r)
	os.Exit(m.Run())
}

func TestScoreHighPainStack(t *testing.T) {
	t.Parallel()

	text := &LeadText{
		Context: "voluum alternative needed. postback failing on FTD every week.",
	}
	p := ScoreText(GetRegistry(), text)
	if p != PriorityHigh {
		t.Fatalf("expected High, got %s score=%d matched=%v", p, text.Score, text.Matched)
	}
	if text.Score < 35 {
		t.Fatalf("score too low: %d", text.Score)
	}
}

func GetRegistry() *Registry {
	return defaultRegistry
}
