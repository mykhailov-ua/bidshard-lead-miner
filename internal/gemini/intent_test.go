package gemini

import (
	"context"
	"testing"
)

func TestClassifyIntent(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"intent":"buyer_search","confidence":0.92,"why":"asks tracker alternative"}`)
	res, err := cl.ClassifyIntent(context.Background(), "Need alternative to voluum for FB traffic")
	if err != nil {
		t.Fatal(err)
	}
	if res.Intent != "buyer_search" {
		t.Fatalf("intent=%q", res.Intent)
	}
	if !res.Accept(0.8) {
		t.Fatalf("expected accept at 0.8, confidence=%v", res.Confidence)
	}
}

func TestClassifyIntentRejectNoise(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"intent":"noise","confidence":0.95,"why":"promo"}`)
	res, err := cl.ClassifyIntent(context.Background(), "subscribe to our signals channel")
	if err != nil {
		t.Fatal(err)
	}
	if res.Accept(0.8) {
		t.Fatal("expected reject for noise")
	}
}

func TestAnalyzeIntentBatch(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"items":[{"id":"a","intent":"buyer_search","confidence":0.9,"why":"buyer"},{"id":"b","intent":"job_offer","confidence":0.88,"why":"hiring"}]}`)
	out, err := cl.AnalyzeIntentBatch(context.Background(), []IntentBatchInput{
		{ID: "a", Source: "forum:test", Text: "which tracker for igaming"},
		{ID: "b", Source: "forum:test", Text: "hiring media buyer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Intent != "buyer_search" || out[1].Intent != "job_offer" {
		t.Fatalf("results=%v", out)
	}
}
