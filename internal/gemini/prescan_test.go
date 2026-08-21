package gemini

import (
	"context"
	"testing"
)

type stubEmbedder struct {
	vectors map[string][]float32
}

func (s stubEmbedder) EmbedText(_ context.Context, text string) ([]float32, error) {
	if vec, ok := s.vectors[text]; ok {
		return vec, nil
	}
	return []float32{0, 0, 1}, nil
}

func TestPrescanPainPass(t *testing.T) {
	t.Parallel()

	painVec := []float32{1, 0, 0}
	spamVec := []float32{0, 1, 0}
	embedder := stubEmbedder{vectors: map[string][]float32{
		"custom pain text": painVec,
	}}

	p := NewPrescanWithAnchors(embedder, [][]float32{painVec}, [][]float32{spamVec}, 0.9, 0.9)
	verdict, err := p.EvaluatePain(context.Background(), "custom pain text")
	if err != nil {
		t.Fatal(err)
	}
	if !verdict.PainMatch {
		t.Fatalf("expected pain match, score=%f", verdict.PainScore)
	}
}

func TestPrescanSpamReject(t *testing.T) {
	t.Parallel()

	painVec := []float32{1, 0, 0}
	spamVec := []float32{0, 1, 0}
	embedder := stubEmbedder{vectors: map[string][]float32{
		"spammy promo": spamVec,
	}}
	p := NewPrescanWithAnchors(embedder, [][]float32{painVec}, [][]float32{spamVec}, 0.9, 0.9)
	_ = embedder

	verdict, err := p.EvaluateSpam(context.Background(), "spammy promo")
	if err != nil {
		t.Fatal(err)
	}
	if !verdict.SpamMatch {
		t.Fatalf("expected spam match, score=%f", verdict.SpamScore)
	}
}

func TestPrescanPainMiss(t *testing.T) {
	t.Parallel()

	painVec := []float32{1, 0, 0}
	spamVec := []float32{0, 1, 0}
	embedder := stubEmbedder{vectors: map[string][]float32{
		"custom pain text":      painVec,
		"unrelated hello world": {0, 0, 1},
	}}
	p := NewPrescanWithAnchors(embedder, [][]float32{painVec}, [][]float32{spamVec}, 0.99, 0.99)
	_ = embedder

	verdict, err := p.EvaluatePain(context.Background(), "unrelated hello world")
	if err != nil {
		t.Fatal(err)
	}
	if verdict.PainMatch {
		t.Fatal("expected no pain match")
	}
}
