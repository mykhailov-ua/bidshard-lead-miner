package gemini

import (
	"context"
	"fmt"
	"sync"
)

// TextEmbedder embeds text for semantic prescan and clustering.
type TextEmbedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

var painAnchors = []string{
	// Fixed affiliate/tracker pain phrases; compared via cosine to incoming text at painMin threshold.
	"voluum alternative tracker migration postback failing self-hosted",
	"keitaro too expensive redtrack binom affiliate ad tracker",
	"media buying team igaming affiliate s2s postback click loss",
	"looking for tracker alternative high spend dedicated infra",
}

var spamAnchors = []string{
	"join our vip signal group subscribe paid mentorship course",
	"free affiliate course dm me for signals limited slots",
	"click the link below telegram channel promo subscribе",
}

// Prescan uses cached anchor embeddings for cheap intent/spam checks.
type Prescan struct {
	embedder TextEmbedder
	painMin  float64
	spamMin  float64

	mu     sync.Mutex
	warmed bool
	pain   [][]float32
	spam   [][]float32
}

type PrescanVerdict struct {
	PainMatch bool
	SpamMatch bool
	PainScore float64
	SpamScore float64
}

func NewPrescan(embedder TextEmbedder, painMin, spamMin float64) *Prescan {
	if painMin <= 0 {
		painMin = 0.78
	}
	if spamMin <= 0 {
		spamMin = 0.82
	}
	return &Prescan{
		embedder: embedder,
		painMin:  painMin,
		spamMin:  spamMin,
	}
}

// NewPrescanWithAnchors wires precomputed vectors (tests).
func NewPrescanWithAnchors(embedder TextEmbedder, pain, spam [][]float32, painMin, spamMin float64) *Prescan {
	p := NewPrescan(embedder, painMin, spamMin)
	p.embedder = embedder
	p.pain = pain
	p.spam = spam
	p.warmed = true
	return p
}

func (p *Prescan) EvaluatePain(ctx context.Context, text string) (PrescanVerdict, error) {
	if p == nil || p.embedder == nil && !p.warmed {
		return PrescanVerdict{}, fmt.Errorf("prescan: not configured")
	}
	if err := p.warm(ctx); err != nil {
		return PrescanVerdict{}, err
	}
	vec, err := p.embedText(ctx, text)
	if err != nil {
		return PrescanVerdict{}, err
	}
	score := maxCosine(vec, p.pain)
	return PrescanVerdict{PainMatch: score >= p.painMin, PainScore: score}, nil
}

func (p *Prescan) EvaluateSpam(ctx context.Context, text string) (PrescanVerdict, error) {
	if p == nil || p.embedder == nil && !p.warmed {
		return PrescanVerdict{}, fmt.Errorf("prescan: not configured")
	}
	if err := p.warm(ctx); err != nil {
		return PrescanVerdict{}, err
	}
	vec, err := p.embedText(ctx, text)
	if err != nil {
		return PrescanVerdict{}, err
	}
	score := maxCosine(vec, p.spam)
	return PrescanVerdict{SpamMatch: score >= p.spamMin, SpamScore: score}, nil
}

func (p *Prescan) warm(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("prescan: nil")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.warmed {
		return nil
	}
	if p.embedder == nil {
		return fmt.Errorf("prescan: embedder required")
	}
	pain, err := embedAnchors(ctx, p.embedder, painAnchors)
	if err != nil {
		return err
	}
	spam, err := embedAnchors(ctx, p.embedder, spamAnchors)
	if err != nil {
		return err
	}
	p.pain = pain
	p.spam = spam
	p.warmed = true
	return nil
}

func (p *Prescan) embedText(ctx context.Context, text string) ([]float32, error) {
	if p.embedder == nil {
		return nil, fmt.Errorf("prescan: embedder required")
	}
	return p.embedder.EmbedText(ctx, text)
}

func embedAnchors(ctx context.Context, embedder TextEmbedder, anchors []string) ([][]float32, error) {
	out := make([][]float32, 0, len(anchors))
	for _, anchor := range anchors {
		vec, err := embedder.EmbedText(ctx, anchor)
		if err != nil {
			return nil, err
		}
		out = append(out, vec)
	}
	return out, nil
}

func maxCosine(vec []float32, anchors [][]float32) float64 {
	var best float64
	for _, anchor := range anchors {
		if sim := CosineSimilarity(vec, anchor); sim > best {
			best = sim
		}
	}
	return best
}
