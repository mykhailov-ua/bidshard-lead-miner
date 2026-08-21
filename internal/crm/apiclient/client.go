package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	parsercfg "github.com/bidshard/parser/internal/config"
)

const defaultTimeout = 15 * time.Second

type Config struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
}

func LoadConfig() (Config, error) {
	parsercfg.LoadDotEnv()
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("CRM_API_URL")), "/")
	if base == "" {
		// Local dev: hit crm-bot run listener without setting CRM_API_URL.
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("CRM_WEBHOOK_ADDR")), "/")
		if base != "" && !strings.HasPrefix(base, "http") {
			base = "http://" + base
		}
	}
	cfg := Config{
		BaseURL:  base,
		Username: strings.TrimSpace(os.Getenv("CRM_API_USER")),
		Password: os.Getenv("CRM_API_PASSWORD"),
		Timeout:  envDuration("CRM_API_TIMEOUT", defaultTimeout),
	}
	if cfg.BaseURL == "" {
		return Config{}, fmt.Errorf("CRM_API_URL empty")
	}
	return cfg, nil
}

type Client struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) GetJSON(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) PostJSON(ctx context.Context, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, body, out)
}

func (c *Client) PatchJSON(ctx context.Context, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPatch, path, body, out)
}

func (c *Client) DeleteJSON(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodDelete, path, nil, out)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	if c == nil {
		return fmt.Errorf("api client not initialized")
	}
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Basic auth when CRM_API_USER is set (Caddy reverse_proxy basicauth on VPS).
	if user := c.cfg.Username; user != "" {
		req.SetBasicAuth(user, c.cfg.Password)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("api %s %s: %s", method, path, msg)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) url(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.cfg.BaseURL + path
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
