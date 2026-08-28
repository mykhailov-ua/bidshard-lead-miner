package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"time"

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
		task.Stats.RejectedGeo.Add(1)
		return
	}
	if outcome.HardRejected {
		task.Stats.HardRejected.Add(1)
		return
	}
	if outcome.Dedup {
		task.Stats.Dedup.Add(1)
		return
	}
	if outcome.DroppedMX {
		task.Stats.Dropped.Add(1)
		return
	}
	if !outcome.Accepted {
		task.Stats.Dropped.Add(1)
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
