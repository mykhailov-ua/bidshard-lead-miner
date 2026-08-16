package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

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
	switch r.mode {
	case "quiet":
		if r.onReport != nil {
			r.onReport()
		}
		return
	case "ndjson":
		r.writeNDJSON(stats)
	default:
		r.writeTable(stats)
	}
	if r.onReport != nil {
		r.onReport()
	}
}

func (r *Reporter) writeNDJSON(stats pipeline.RoundStats) {
	enc := json.NewEncoder(r.stdout)
	for _, lead := range stats.Leads {
		_ = enc.Encode(sink.LeadJSONMap(lead))
	}
}

func (r *Reporter) writeTable(stats pipeline.RoundStats) {
	totalSources := stats.SourcesOK + stats.SourcesFail
	fmt.Fprintf(r.stdout, "scan round %s  duration=%s  sources=%d/%d ok\n",
		stats.RoundID,
		stats.Duration.Round(time.Millisecond*100),
		stats.SourcesOK,
		totalSources,
	)
	fmt.Fprintf(r.stdout, "  raw=%d  accepted=%d  rejected_geo=%d  dedup=%d  dropped=%d\n",
		stats.RawTotal, stats.Accepted, stats.RejectedGeo, stats.Dedup, stats.Dropped,
	)
	fmt.Fprintf(r.stdout, "  priority: high=%d  medium=%d  low=%d\n\n",
		stats.High, stats.Medium, stats.Low,
	)

	for _, lead := range stats.Leads {
		if lead.Priority != "High" {
			continue
		}
		matched := strings.Join(lead.Matched, " · ")
		contacts := strings.Join(lead.Contacts, "  ")
		fmt.Fprintf(r.stdout, "  HIGH  score=%d  %s\n", lead.Score, matched)
		fmt.Fprintf(r.stdout, "        %s  %s\n", contacts, lead.Source)
		if lead.Snippet != "" {
			fmt.Fprintf(r.stdout, "        «%s»\n", lead.Snippet)
		}
		fmt.Fprintln(r.stdout)
	}
}
