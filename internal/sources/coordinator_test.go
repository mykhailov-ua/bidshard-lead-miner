package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

func TestCoordinatorCancelsPreviousRound(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		WorkerCount:       1,
		TaskBuffer:        4,
		SourceConcurrency: 1,
		ScanTimeout:       5 * time.Second,
		HTTPTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	}

	blocking := NewBlockingStubSource("stub:blocking")
	coordinator := NewCoordinator(cfg, []Source{blocking})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	taskCh := make(chan pipeline.Task, 4)
	scanCh := make(chan struct{}, 1)
	statsCh := make(chan pipeline.RoundStats, 4)

	var wg sync.WaitGroup
	coordinator.Run(ctx, &wg, scanCh, taskCh, statsCh)

	scanCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)

	roundCancel := coordinator.ActiveRoundCancel()
	if roundCancel == nil {
		t.Fatal("expected active round cancel function")
	}

	scanCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		roundCancel()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("previous round cancel did not return after new scan")
	}

	cancel()
	wg.Wait()
}

func TestCoordinatorCollectUsesScanTimeout(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		WorkerCount:       1,
		TaskBuffer:        8,
		SourceConcurrency: 1,
		ScanTimeout:       400 * time.Millisecond,
		HTTPTimeout:       50 * time.Millisecond,
	}

	slow := &StubSource{
		name:  "stub:slow",
		delay: 150 * time.Millisecond,
		items: []model.RawItem{{Raw: "voluum alternative", Contact: "x@y.com"}},
	}
	coordinator := NewCoordinator(cfg, []Source{slow})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	taskCh := make(chan pipeline.Task, 8)
	statsCh := make(chan pipeline.RoundStats, 1)

	var drainWG sync.WaitGroup
	drainWG.Add(1)
	go func() {
		defer drainWG.Done()
		for task := range taskCh {
			task.Stats.FinishTask()
		}
	}()

	coordinator.runRound(ctx, taskCh, statsCh)
	close(taskCh)
	drainWG.Wait()

	stats := <-statsCh
	if stats.SourcesFail != 0 {
		t.Fatalf("sources_fail=%d, want 0 (collect should outlive HTTPTimeout)", stats.SourcesFail)
	}
	if stats.SourcesOK != 1 {
		t.Fatalf("sources_ok=%d, want 1", stats.SourcesOK)
	}
}

func TestCoordinatorDropsOnFullTaskChannel(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		TaskBuffer:        0,
		SourceConcurrency: 1,
		ScanTimeout:       time.Second,
		HTTPTimeout:       time.Second,
	}

	coordinator := NewCoordinator(cfg, []Source{
		NewStubSource("stub:test", []model.RawItem{
			{Raw: "voluum alternative", Contact: "x@y.com"},
		}),
	})
	taskCh := make(chan pipeline.Task)
	state := &pipeline.RoundState{}

	err := coordinator.emit(context.Background(), "abc123", taskCh, state, model.RawItem{
		Source:  "stub:test",
		Raw:     "test",
		Contact: "x@y.com",
	})
	if err != nil {
		t.Fatalf("emit returned %v", err)
	}
	if state.Dropped.Load() != 1 {
		t.Fatalf("dropped=%d, want 1", state.Dropped.Load())
	}
}

func TestCoordinatorRoundStatsRejects(t *testing.T) {
	t.Parallel()

	if err := validate.LoadBlacklistDomains("../../data/blacklist_domains.txt"); err != nil {
		t.Fatalf("load blacklist: %v", err)
	}
	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	proc := &pipeline.Processor{
		Registry:              reg,
		Seen:                  dedup.NewSeenCache(1000, 0),
		Store:                 sink.NewStubStore(),
		MX:                    validate.StubMX{OK: true},
		LanderOutreachEnabled: false,
	}

	cfg := config.Config{
		WorkerCount:       2,
		TaskBuffer:        8,
		SourceConcurrency: 2,
		ScanTimeout:       5 * time.Second,
		HTTPTimeout:       time.Second,
		ProcessorTaskTimeout: 5 * time.Second,
	}
	coordinator := NewCoordinator(cfg, []Source{
		NewStubSource("lander:voluum.com", []model.RawItem{
			{Raw: "voluum alternative postback failing", Contact: "email:support@voluum.com"},
		}),
		NewStubSource("github:keitaroinc/docker-ckan", []model.RawItem{
			{Raw: "voluum postback failing s2s migration", Contact: "github:keitaroinc"},
		}),
		NewStubSource("telegram:@igaming_news", []model.RawItem{
			{Raw: "daily igaming news digest", Contact: "@igaming_news", ChatType: "channel"},
		}),
		NewStubSource("lander:example-tracker.com", []model.RawItem{
			{Raw: "voluum alternative postback failing", Contact: "email:ops@example-tracker.com"},
		}),
		NewStubSource("stub:test", []model.RawItem{
			{Raw: "generic unrelated text no tracker keywords", Contact: "email:ops@example.com"},
		}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	taskCh := make(chan pipeline.Task, 8)
	statsCh := make(chan pipeline.RoundStats, 1)

	var wg sync.WaitGroup
	pool := pipeline.NewPool(cfg.WorkerCount, proc, cfg.ProcessorTaskTimeout)
	pool.Run(ctx, &wg, taskCh)

	go func() {
		coordinator.runRound(ctx, taskCh, statsCh)
		close(taskCh)
	}()

	stats := <-statsCh
	wg.Wait()

	if stats.RawTotal != 5 {
		t.Fatalf("raw_total=%d want 5", stats.RawTotal)
	}
	if stats.RejectedBlacklist != 1 {
		t.Fatalf("rejected_blacklist=%d want 1", stats.RejectedBlacklist)
	}
	if stats.RejectedGitHubVendor != 1 {
		t.Fatalf("rejected_github_vendor=%d want 1", stats.RejectedGitHubVendor)
	}
	if stats.RejectedTelegramSpam != 0 {
		t.Fatalf("rejected_telegram_spam=%d want 0", stats.RejectedTelegramSpam)
	}
	if stats.RejectedIntelOnly != 2 {
		t.Fatalf("rejected_intel_only=%d want 2", stats.RejectedIntelOnly)
	}
	if stats.RejectedLowPriority != 1 {
		t.Fatalf("rejected_low_priority=%d want 1", stats.RejectedLowPriority)
	}
	if stats.Accepted != 0 {
		t.Fatalf("accepted=%d want 0", stats.Accepted)
	}
}
