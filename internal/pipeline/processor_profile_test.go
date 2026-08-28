package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/enrich"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/validate"
)

func TestProcessorProfileEnrichForumAccepts(t *testing.T) {
	t.Parallel()

	html := `<html><body><p>voluum alternative postback failing. Email ops@igaming-team.com</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	fetcher := forum.NewFetcher(5*time.Second, server.URL)
	pe := enrich.NewProfileEnricherForTest(fetcher, server.Client(), "")

	proc := &Processor{
		Registry:        reg,
		Seen:            dedup.NewSeenCache(10, 0),
		Store:           sink.NewStubStore(),
		MX:              validate.StubMX{OK: true},
		ProfileEnricher: pe,
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:      "forum:affiliatefix.com/voluum-thread",
			Raw:         "voluum alternative postback failing on renewal",
			Contact:     "forum:user/media_buyer",
			Username:    "media_buyer",
			ForumUserID: "12",
		},
	})
	if !out.Accepted {
		t.Fatal("expected forum profile enrich to supply email and accept")
	}
}
