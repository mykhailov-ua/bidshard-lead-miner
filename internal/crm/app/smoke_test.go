package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/crm/webhook"
	"github.com/bidshard/parser/internal/sink"
)

func TestCRMBotWebhookSmoke(t *testing.T) {
	handler := webhook.NewHandler("smoke-secret")
	srv := httptest.NewServer(handler)
	defer srv.Close()

	body, err := json.Marshal(sink.LeadDoc{
		HashID:  "abcdef1234567890abcdef1234567890",
		Score:   90,
		Source:  "forum:smoke",
		Status:  "new",
		Snippet: "smoke test lead",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/leads", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer smoke-secret")
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("webhook status=%d", resp.StatusCode)
	}
}
