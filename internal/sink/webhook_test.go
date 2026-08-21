package sink

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestWebhookClientPostsLeadJSON(t *testing.T) {
	var mu sync.Mutex
	var got LeadDoc

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("auth header=%q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		mu.Lock()
		defer mu.Unlock()
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewWebhookClient(srv.URL, "test-secret", 2*time.Second)
	lead := model.Lead{
		HashID:   "abc123",
		Source:   "forum",
		Priority: "High",
		Score:    80,
	}
	client.post(lead)

	mu.Lock()
	defer mu.Unlock()
	if got.HashID != "abc123" {
		t.Fatalf("hash_id=%q", got.HashID)
	}
}

func TestWrapWebhookUpsertNotifies(t *testing.T) {
	done := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := NewStubStore()
	store := WrapWebhook(inner, srv.URL, "", time.Second)
	if err := store.Upsert(context.Background(), model.Lead{HashID: "x1", Source: "forum"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("webhook not called")
	}
}

func TestStoreNotifiesCRM(t *testing.T) {
	t.Parallel()

	inner := NewStubStore()
	if StoreNotifiesCRM(inner) {
		t.Fatal("stub store should not notify")
	}
	wrapped := WrapWebhook(inner, "http://127.0.0.1:1", "", time.Second)
	if !StoreNotifiesCRM(wrapped) {
		t.Fatal("webhook store should notify")
	}
	bulk := NewBulkStore(wrapped, 10, time.Second)
	if !StoreNotifiesCRM(bulk) {
		t.Fatal("bulk over webhook should notify")
	}
	bulkOnly := NewBulkStore(inner, 10, time.Second)
	if StoreNotifiesCRM(bulkOnly) {
		t.Fatal("bulk without webhook should not notify")
	}
}

func TestValidateWebhookURL(t *testing.T) {
	if err := ValidateWebhookURL(""); err == nil {
		t.Fatal("expected error for empty url")
	}
	if err := ValidateWebhookURL("https://crm.example/hook"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWebhookURL("ftp://x"); err == nil {
		t.Fatal("expected scheme error")
	}
}
