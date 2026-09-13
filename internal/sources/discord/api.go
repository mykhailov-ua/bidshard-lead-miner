package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bidshard/parser/internal/httpclient"
)

type API struct {
	pool    *TokenPool
	client  *http.Client
	baseURL string
}

func NewAPI(pool *TokenPool, client *http.Client, baseURL string) *API {
	if baseURL == "" {
		baseURL = apiBase
	}
	return &API{pool: pool, client: client, baseURL: baseURL}
}

type invitePreview struct {
	Code    string `json:"code"`
	Guild   *guild `json:"guild"`
	Channel *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type int    `json:"type"`
	} `json:"channel"`
}

type guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type guildChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

func (a *API) GetInvite(ctx context.Context, code string) (*invitePreview, error) {
	url := fmt.Sprintf("%s/invites/%s?with_counts=true", a.baseURL, code)
	var out invitePreview
	if err := a.doJSON(ctx, http.MethodGet, url, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *API) AcceptInvite(ctx context.Context, code string) (*invitePreview, error) {
	url := fmt.Sprintf("%s/invites/%s", a.baseURL, code)
	var out invitePreview
	if err := a.doJSON(ctx, http.MethodPost, url, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *API) ListGuildChannels(ctx context.Context, guildID string) ([]guildChannel, error) {
	url := fmt.Sprintf("%s/guilds/%s/channels", a.baseURL, guildID)
	var out []guildChannel
	if err := a.doJSON(ctx, http.MethodGet, url, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *API) ListMyGuilds(ctx context.Context) ([]guild, error) {
	url := fmt.Sprintf("%s/users/@me/guilds?limit=200", a.baseURL)
	var out []guild
	if err := a.doJSON(ctx, http.MethodGet, url, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *API) doJSON(ctx context.Context, method, url string, body []byte, out any) error {
	attempts := a.pool.Len()
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		token := a.pool.Pick()
		if token == "" {
			return fmt.Errorf("discord token pool empty")
		}
		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return err
		}
		raw, status, err := a.doAuth(ctx, req, token)
		if err != nil {
			lastErr = err
			a.pool.Rotate()
			continue
		}
		switch status {
		case http.StatusOK, http.StatusCreated:
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(raw, out); err != nil {
				return err
			}
			return nil
		case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
			lastErr = fmt.Errorf("discord http %d", status)
			a.pool.Rotate()
			continue
		default:
			return fmt.Errorf("discord http %d", status)
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("discord api failed")
}

func (a *API) doAuth(ctx context.Context, req *http.Request, token string) ([]byte, int, error) {
	for _, auth := range []string{"Bot " + token, token} {
		cloned := req.Clone(ctx)
		cloned.Header.Set("Authorization", auth)
		cloned.Header.Set("Content-Type", "application/json")
		body, status, err := httpclient.DoBytes(a.client, cloned, 2<<20)
		if err != nil {
			return nil, 0, err
		}
		if status != http.StatusUnauthorized {
			return body, status, nil
		}
	}
	return nil, http.StatusUnauthorized, fmt.Errorf("discord unauthorized")
}
