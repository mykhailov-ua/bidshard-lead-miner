package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/model"
)

// WebhookClient POSTs lead JSON to an external CRM endpoint (fire-and-forget).
type WebhookClient struct {
	url        string
	secret     string
	httpClient *http.Client
}

func NewWebhookClient(url, secret string, timeout time.Duration) *WebhookClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookClient{
		url:    strings.TrimSpace(url),
		secret: strings.TrimSpace(secret),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *WebhookClient) NotifyLead(lead model.Lead) {
	if c == nil || c.url == "" {
		return
	}
	// Fire-and-forget: never block accept path on CRM latency or failures.
	go c.post(lead)
}

func (c *WebhookClient) post(lead model.Lead) {
	doc := LeadExport(lead)
	body, err := json.Marshal(doc)
	if err != nil {
		slog.Warn("crm webhook marshal failed", "error", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		slog.Warn("crm webhook request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("crm webhook post failed", "hash_id", lead.HashID, "error", err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("crm webhook non-2xx", "hash_id", lead.HashID, "status", resp.StatusCode)
	}
}

type webhookStore struct {
	inner  Store
	notify *WebhookClient
}

// WrapWebhook wraps a Store and notifies CRM after successful upsert.
func WrapWebhook(inner Store, url, secret string, timeout time.Duration) Store {
	if inner == nil || strings.TrimSpace(url) == "" {
		return inner
	}
	return &webhookStore{
		inner:  inner,
		notify: NewWebhookClient(url, secret, timeout),
	}
}

func (s *webhookStore) Exists(ctx context.Context, hashID string) (bool, error) {
	return s.inner.Exists(ctx, hashID)
}

func (s *webhookStore) Upsert(ctx context.Context, lead model.Lead) error {
	if err := s.inner.Upsert(ctx, lead); err != nil {
		return err
	}
	s.notify.NotifyLead(lead)
	return nil
}

func (s *webhookStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	return s.inner.UpdateStatus(ctx, hashID, status)
}

var _ Store = (*webhookStore)(nil)

func ValidateWebhookURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("PARSER_CRM_WEBHOOK_URL empty")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return fmt.Errorf("PARSER_CRM_WEBHOOK_URL must start with http:// or https://")
	}
	return nil
}
