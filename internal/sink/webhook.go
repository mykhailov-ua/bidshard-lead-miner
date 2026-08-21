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

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/model"
)

// WebhookClient POSTs lead JSON to an external CRM endpoint (fire-and-forget).
type WebhookClient struct {
	url        string
	secret     string
	httpClient *http.Client
	heatGate   WebhookHeatGate
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
		heatGate: WebhookHeatGate{MinTier: entity.HeatTierCold},
	}
}

// WithHeatMin sets minimum heat_tier for webhook delivery (cold = no gate).
func (c *WebhookClient) WithHeatMin(minTier string) *WebhookClient {
	if c == nil {
		return c
	}
	c.heatGate.MinTier = strings.TrimSpace(minTier)
	return c
}

func (c *WebhookClient) NotifyLead(lead model.Lead) {
	if c == nil || c.url == "" {
		return
	}
	if !c.allowsHeat(lead) {
		c.logHeatSkip(lead)
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

// AttachWebhook wraps a Store with an existing WebhookClient (shared with warm path in defer mode).
func AttachWebhook(inner Store, notify *WebhookClient) Store {
	if inner == nil || notify == nil {
		return inner
	}
	return &webhookStore{
		inner:  inner,
		notify: notify,
	}
}

// WrapWebhook wraps a Store and notifies CRM after successful upsert.
func WrapWebhook(inner Store, url, secret string, timeout time.Duration) Store {
	if inner == nil || strings.TrimSpace(url) == "" {
		return inner
	}
	return AttachWebhook(inner, NewWebhookClient(url, secret, timeout))
}

// StoreNotifiesCRM reports whether Upsert on s triggers a hot-path CRM webhook.
// Only unwraps BulkStore -> webhookStore; other wrapper types return false.
func StoreNotifiesCRM(s Store) bool {
	for s != nil {
		switch v := s.(type) {
		case *webhookStore:
			return true
		case *BulkStore:
			s = v.UnderlyingStore()
		default:
			return false
		}
	}
	return false
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
