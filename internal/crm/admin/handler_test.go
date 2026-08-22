package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerPurgeConfirmRequired(t *testing.T) {
	h := NewHandler(nil, nil)
	body := []byte(`{"all":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/leads/purge", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHandlerListRequiresStore(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/leads", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}
