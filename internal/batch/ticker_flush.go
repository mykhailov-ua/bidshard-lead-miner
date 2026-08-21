package batch

import (
	"context"
	"time"
)

// RunTickerFlush batches items from in until ctx is done or in closes.
// flush errors are ignored here; callers log inside flush (cold-path junk ingest).
func RunTickerFlush[T any](
	ctx context.Context,
	in <-chan T,
	batchSize int,
	interval time.Duration,
	flush func([]T) error,
) {
	if in == nil {
		return
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}

	buf := make([]T, 0, batchSize)
	flushBuf := func() {
		if len(buf) == 0 {
			return
		}
		batch := append([]T(nil), buf...)
		buf = buf[:0]
		_ = flush(batch)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flushBuf()
			return
		case <-ticker.C:
			flushBuf()
		case item, ok := <-in:
			if !ok {
				flushBuf()
				return
			}
			buf = append(buf, item)
			if len(buf) >= batchSize {
				flushBuf()
			}
		}
	}
}
