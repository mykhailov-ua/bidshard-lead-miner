package output

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/sink"
)

type Reporter struct {
	mode     string
	stdout   io.Writer
	onReport func()
}

func NewReporter(mode string, stdout io.Writer) *Reporter {
	if stdout == nil {
		stdout = os.Stdout
	}
	return &Reporter{mode: strings.ToLower(mode), stdout: stdout}
}

func (r *Reporter) SetOnReport(fn func()) {
	r.onReport = fn
}

func (r *Reporter) Run(ctx context.Context, wg *sync.WaitGroup, statsCh <-chan pipeline.RoundStats) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.loop(ctx, statsCh)
	}()
}

func (r *Reporter) loop(ctx context.Context, statsCh <-chan pipeline.RoundStats) {
	for {
		select {
		case <-ctx.Done():
			return
		case stats, ok := <-statsCh:
			if !ok {
				return
			}
			r.handle(stats)
		}
	}
}

func (r *Reporter) handle(stats pipeline.RoundStats) {
	switch resolveOutputMode(r.mode, r.stdout) {
	case "quiet":
		if r.onReport != nil {
			r.onReport()
		}
		return
	case "ndjson":
		r.writeNDJSON(stats)
	case "json-pretty":
		r.writeJSONPretty(stats)
	default:
		r.writePretty(stats)
	}
	if r.onReport != nil {
		r.onReport()
	}
}

func (r *Reporter) writeNDJSON(stats pipeline.RoundStats) {
	for _, lead := range stats.Leads {
		_ = sink.EncodeLeadExport(r.stdout, lead, sink.ExportFormatNDJSON)
	}
}
