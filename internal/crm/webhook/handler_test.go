package webhook_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/crm/webhook"
	"github.com/bidshard/parser/internal/sink"
)

func TestHandlerAcceptsLead(t *testing.T) {
	handler := webhook.NewHandler("secret")
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

func TestHandlerUnauthorized(t *testing.T) {
	handler := webhook.NewHandler("secret")
	req := httptest.NewRequest(http.MethodPost, "/v1/leads", strings.NewReader(`{"hash_id":"x"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlerBadRequest(t *testing.T) {
	handler := webhook.NewHandler("")
	req := httptest.NewRequest(http.MethodPost, "/v1/leads", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
