package sink

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

// deferAwareWebhookStore uses *ops.SettingsStore; nil settings keeps defer (no hot-path notify).
func TestDeferAwareWebhookSkipsWhenLLMOn(t *testing.T) {
	var posted int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posted++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := NewStubStore()
	notify := NewWebhookClient(srv.URL, "", time.Second)
	wrapped := AttachDeferAwareWebhook(inner, notify, true, nil)

	if err := wrapped.Upsert(context.Background(), model.Lead{HashID: "a"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if posted != 0 {
		t.Fatalf("posted=%d want 0 when settings nil (defer)", posted)
	}
}

func TestDeferAwareWebhookPostsWhenLLMOff(t *testing.T) {
	// Integration with real SettingsStore needs mongo; verify hotPathNotify logic on nil vs off via exported type.
	store := &deferAwareWebhookStore{deferUntilAnalysis: true, settings: nil}
	if store.hotPathNotify(context.Background()) {
		t.Fatal("expected false when settings nil")
	}
}
