package reviews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestReviewsCrawlerHTML(t *testing.T) {
	fixture := `<html><body>
	<article class="review-card">
		<span class="consumer-name">DisappointedUser</span>
		<div data-service-review-rating="1"></div>
		<p class="review-text">Voluum pricing is far too expensive for small teams. Postback failed twice last week. Contact me telegram:@disappointed_user</p>
	</article>
	<article class="review-card">
		<span class="consumer-name">HappyUser</span>
		<div data-service-review-rating="5"></div>
		<p class="review-text">Great software love it!</p>
	</article>
	</body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fixture))
	}))
	defer ts.Close()

	cfg := config.Config{
		ForumBaseURL: ts.URL,
	}

	crawler := NewCrawler(cfg, ts.Client())
	crawler.SetBaseURL(ts.URL)

	var emitted []model.RawItem
	err := crawler.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		emitted = append(emitted, item)
		return nil
	})

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(emitted) == 0 {
		t.Fatalf("expected emitted items for 1-star review, got 0")
	}
	if emitted[0].ContactTelegram() != "telegram:@disappointed_user" {
		t.Errorf("got contact %q, want telegram:@disappointed_user", emitted[0].ContactTelegram())
	}
}
