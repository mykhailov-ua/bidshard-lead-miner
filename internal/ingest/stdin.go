package ingest

import (
	"bufio"
	"context"
	"io"
	"log/slog"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
)

func Scan(ctx context.Context, r io.Reader, taskCh chan<- pipeline.Task, stats *pipeline.RoundState, roundID string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var item model.RawItem
		var err error
		item, err = parseNDJSONLine(scanner.Bytes())
		if err != nil {
			slog.Warn("ingest skip bad json", "error", err)
			continue
		}
		if err := validateTelegramItem(item); err != nil {
			slog.Warn("ingest skip invalid telegram item", "error", err, "source", item.Source)
			continue
		}
		emit(ctx, taskCh, stats, roundID, item)
	}
}

func RunStdin(ctx context.Context, wg ingestWaitGroup, r io.Reader, taskCh chan<- pipeline.Task, stats *pipeline.RoundState, roundID string) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		Scan(ctx, r, taskCh, stats, roundID)
	}()
}

type ingestWaitGroup interface {
	Add(int)
	Done()
}

func emit(ctx context.Context, taskCh chan<- pipeline.Task, stats *pipeline.RoundState, roundID string, item model.RawItem) {
	task := pipeline.Task{
		RoundID: roundID,
		Item:    item,
		Stats:   stats,
	}
	if err := pipeline.TryEnqueue(ctx, taskCh, task); err != nil && ctx.Err() == nil {
		slog.Warn("task channel full",
			"round_id", roundID,
			"source", item.Source,
			"contact", item.MaskedContact(),
		)
	}
}
