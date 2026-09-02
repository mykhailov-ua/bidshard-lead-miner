package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/scoring"
)

type Pool struct {
	workerCount int
	processor   *Processor
	taskTimeout time.Duration
}

func NewPool(workerCount int, processor *Processor, taskTimeout time.Duration) *Pool {
	if taskTimeout <= 0 {
		taskTimeout = 90 * time.Second
	}
	return &Pool{
		workerCount: workerCount,
		processor:   processor,
		taskTimeout: taskTimeout,
	}
}

func (p *Pool) Run(ctx context.Context, wg *sync.WaitGroup, tasks <-chan Task) {
	for i := 0; i < p.workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p.worker(ctx, id, tasks)
		}(i + 1)
	}
}

func (p *Pool) worker(ctx context.Context, id int, tasks <-chan Task) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			p.process(ctx, id, task)
		}
	}
}

func (p *Pool) process(ctx context.Context, id int, task Task) {
	if task.Stats != nil {
		defer task.Stats.FinishTask()
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	if task.Stats != nil {
		task.Stats.RawTotal.Add(1)
	}

	if p.processor == nil {
		return
	}

	taskCtx, cancel := context.WithTimeout(ctx, p.taskTimeout)
	defer cancel()

	outcome := p.processor.Process(taskCtx, task)
	if task.Stats == nil {
		return
	}

	if outcome.RejectedGeo {
		recordRoundReject(task.Stats, "geo")
		return
	}
	if outcome.HardRejected {
		recordRoundReject(task.Stats, "hard_reject")
		return
	}
	if outcome.Dedup {
		recordRoundReject(task.Stats, "dedup")
		return
	}
	if outcome.DroppedMX {
		recordRoundReject(task.Stats, "mx")
		return
	}
	if !outcome.Accepted {
		reason := outcome.RejectReason
		if reason == "" {
			reason = "dropped"
		}
		recordRoundReject(task.Stats, reason)
		return
	}

	task.Stats.Accepted.Add(1)
	switch outcome.Lead.Priority {
	case string(scoring.PriorityHigh):
		task.Stats.High.Add(1)
	case string(scoring.PriorityMedium):
		task.Stats.Medium.Add(1)
	default:
		task.Stats.Low.Add(1)
	}
	task.Stats.AddLead(outcome.Lead)

	slog.Debug("lead accepted",
		"worker", id,
		"round_id", task.RoundID,
		"priority", outcome.Lead.Priority,
		"score", outcome.Lead.Score,
	)
}

func recordRoundReject(stats *RoundState, reason string) {
	if stats == nil {
		return
	}
	switch reason {
	case "geo":
		stats.RejectedGeo.Add(1)
	case "hard_reject":
		stats.HardRejected.Add(1)
	case "dedup":
		stats.Dedup.Add(1)
	case "mx":
		stats.RejectedMX.Add(1)
	case "blacklist":
		stats.RejectedBlacklist.Add(1)
	case "intel_only":
		stats.RejectedIntelOnly.Add(1)
	case "lander_no_buyer_signal":
		stats.RejectedLanderNoBuyer.Add(1)
	case "github_vendor":
		stats.RejectedGitHubVendor.Add(1)
	case "telegram_spam":
		stats.RejectedTelegramSpam.Add(1)
	case "low_priority":
		stats.RejectedLowPriority.Add(1)
	case "icp":
		stats.RejectedICP.Add(1)
	case "intent":
		stats.RejectedIntent.Add(1)
	case "lang":
		stats.RejectedLang.Add(1)
	case "context":
		stats.RejectedContext.Add(1)
	case "contact":
		stats.RejectedContact.Add(1)
	case "no_contacts":
		stats.RejectedNoContacts.Add(1)
	case "email_no_context":
		stats.RejectedEmailNoContext.Add(1)
	case "role_email":
		stats.RejectedRoleEmail.Add(1)
	case "empty_hash":
		stats.RejectedEmptyHash.Add(1)
	default:
		stats.Dropped.Add(1)
		reason = "dropped"
	}
	metrics.RecordProcessorReject(reason)
}
