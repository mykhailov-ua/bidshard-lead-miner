package pipeline

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type RoundState struct {
	RawTotal    atomic.Int64
	Accepted    atomic.Int64
	RejectedGeo atomic.Int64
	Dedup       atomic.Int64
	Dropped     atomic.Int64
	High        atomic.Int64
	Medium      atomic.Int64
	Low         atomic.Int64
	SourcesOK   atomic.Int64
	SourcesFail atomic.Int64

	mu    sync.Mutex
	leads []model.Lead
	wg    sync.WaitGroup
}

func (s *RoundState) Wait() {
	s.wg.Wait()
}

func (s *RoundState) TrackTask() {
	s.wg.Add(1)
}

func (s *RoundState) FinishTask() {
	s.wg.Done()
}

func (s *RoundState) AddLead(lead model.Lead) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leads = append(s.leads, lead)
}

func (s *RoundState) Leads() []model.Lead {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Lead, len(s.leads))
	copy(out, s.leads)
	return out
}

type RoundStats struct {
	RoundID     string
	Duration    time.Duration
	SourcesOK   int
	SourcesFail int
	RawTotal    int
	Accepted    int
	RejectedGeo int
	Dedup       int
	Dropped     int
	High        int
	Medium      int
	Low         int
	Leads       []model.Lead
}

func (s *RoundState) Snapshot(roundID string, duration time.Duration) RoundStats {
	return RoundStats{
		RoundID:     roundID,
		Duration:    duration,
		SourcesOK:   int(s.SourcesOK.Load()),
		SourcesFail: int(s.SourcesFail.Load()),
		RawTotal:    int(s.RawTotal.Load()),
		Accepted:    int(s.Accepted.Load()),
		RejectedGeo: int(s.RejectedGeo.Load()),
		Dedup:       int(s.Dedup.Load()),
		Dropped:     int(s.Dropped.Load()),
		High:        int(s.High.Load()),
		Medium:      int(s.Medium.Load()),
		Low:         int(s.Low.Load()),
		Leads:       s.Leads(),
	}
}
