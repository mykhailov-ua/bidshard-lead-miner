package coldpath

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

// AcceptLeadSampler loads random accepted leads for audit.
type AcceptLeadSampler interface {
	SampleRandomAccepts(ctx context.Context, since time.Time, limit int) ([]sink.LeadDoc, error)
}

// GeoClassifier runs deferred geo compliance checks.
type GeoClassifier interface {
	ClassifyGeo(ctx context.Context, text string, contacts []string, blockedCountries []string) (gemini.GeoResult, error)
}

// GeoAuditRunner samples accepts and junk for weekly geo slip checks.
type GeoAuditRunner struct {
	interval time.Duration
	sampleN  int
	junk     *sink.JunkStore
	leads    AcceptLeadSampler
	geo      GeoClassifier
	reports  *sink.GeoAuditStore
	blocked  []string
}

func NewGeoAuditRunner(interval time.Duration, sampleN int, junk *sink.JunkStore, leads AcceptLeadSampler, geo GeoClassifier, reports *sink.GeoAuditStore, blocked []string) *GeoAuditRunner {
	interval = worker.DurationOr(interval, 7*24*time.Hour)
	if sampleN <= 0 {
		sampleN = 10
	}
	if len(blocked) == 0 {
		blocked = []string{"RU", "BY"}
	}
	return &GeoAuditRunner{
		interval: interval,
		sampleN:  sampleN,
		junk:     junk,
		leads:    leads,
		geo:      geo,
		reports:  reports,
		blocked:  blocked,
	}
}

func (r *GeoAuditRunner) Run(ctx context.Context) {
	if r == nil || r.junk == nil || r.leads == nil || r.geo == nil || r.reports == nil {
		return
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	r.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *GeoAuditRunner) runOnce(ctx context.Context) {
	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	accepts, err := r.leads.SampleRandomAccepts(ctx, since, r.sampleN)
	if err != nil {
		slog.Warn("geo audit accept sample failed", "error", err)
		return
	}
	junkRows, err := r.junk.SampleRandom(ctx, since, r.sampleN)
	if err != nil {
		slog.Warn("geo audit junk sample failed", "error", err)
		return
	}

	// Slip: accepted lead that deferred geo would still block; miss: junk row geo would allow.
	acceptSlips := 0
	for _, doc := range accepts {
		res, err := r.geo.ClassifyGeo(ctx, doc.Snippet, nil, r.blocked)
		if err != nil {
			continue
		}
		if res.ShouldReject(r.blocked) {
			acceptSlips++
		}
	}
	junkMisses := 0
	for _, doc := range junkRows {
		if doc.Reason != ReasonGeoReject && doc.Reason != ReasonGeoGeminiReject {
			continue
		}
		res, err := r.geo.ClassifyGeo(ctx, doc.Snippet, nil, r.blocked)
		if err != nil {
			continue
		}
		if !res.ShouldReject(r.blocked) {
			junkMisses++
		}
	}

	slipRate := 0.0
	if len(accepts) > 0 {
		slipRate = float64(acceptSlips) / float64(len(accepts))
	}
	doc := sink.GeoAuditDoc{
		TS:           time.Now().UTC(),
		AcceptSample: len(accepts),
		JunkSample:   len(junkRows),
		AcceptSlips:  acceptSlips,
		JunkMisses:   junkMisses,
		SlipRate:     slipRate,
	}
	if err := r.reports.Insert(ctx, doc); err != nil {
		slog.Warn("geo audit insert failed", "error", err)
		return
	}
	slog.Info("geo audit complete",
		"accept_slips", acceptSlips,
		"junk_misses", junkMisses,
		"slip_rate", slipRate,
	)
}
