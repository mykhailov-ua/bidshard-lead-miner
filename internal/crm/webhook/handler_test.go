package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/crm/webhook"
	"github.com/bidshard/parser/internal/sink"
)

type stubNotifier struct {
	doc sink.LeadDoc
}

func (s *stubNotifier) NotifyLead(_ context.Context, doc sink.LeadDoc) {
	s.doc = doc
}

func TestHandlerAcceptsLead(t *testing.T) {
	handler := webhook.NewHandler("secret", nil)
	body, err := json.Marshal(sink.LeadDoc{HashID: "abc123", Score: 90, Source: "forum:test"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/leads", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHandlerNotifiesLead(t *testing.T) {
	notify := &stubNotifier{}
	handler := webhook.NewHandler("secret", notify)
	body, err := json.Marshal(sink.LeadDoc{HashID: "notify123", Score: 55, Source: "telegram:@voluum"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/leads", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d", rec.Code)
	}
	if notify.doc.HashID != "notify123" {
		t.Fatalf("notify doc=%q", notify.doc.HashID)
	}
}

func TestHandlerUnauthorized(t *testing.T) {
	handler := webhook.NewHandler("secret", nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/leads", strings.NewReader(`{"hash_id":"x"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlerBadRequest(t *testing.T) {
	handler := webhook.NewHandler("", nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/leads", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
