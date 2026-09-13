package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const telegramAPIBase = "https://api.telegram.org"

type Client struct {
	token  string
	http   *http.Client
	apiURL string
}

func NewClient(token string) *Client {
	token = strings.TrimSpace(token)
	return &Client{
		token:  token,
		apiURL: telegramAPIBase + "/bot" + token,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSec int) ([]Update, error) {
	q := url.Values{}
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	q.Set("timeout", fmt.Sprintf("%d", timeoutSec))
	q.Set("allowed_updates", `["message"]`)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/getUpdates?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getUpdates transport: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := readBodyLimited(resp.Body, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("telegram getUpdates read body: %w", err)
	}
	env, err := decodeEnvelope(body)
	if err != nil {
		return nil, fmt.Errorf("telegram getUpdates %w (http=%d)", err, resp.StatusCode)
	}
	if ae := apiErrorFromEnvelope("getUpdates", resp.StatusCode, env); ae != nil {
		return nil, ae
	}
	if resp.StatusCode/100 != 2 {
		return nil, &APIError{
			Method:      "getUpdates",
			HTTPStatus:  resp.StatusCode,
			Description: truncateForLog(string(body), 200),
		}
	}

	var updates []Update
	if len(env.Result) > 0 && string(env.Result) != "null" {
		if err := json.Unmarshal(env.Result, &updates); err != nil {
			return nil, fmt.Errorf("telegram getUpdates decode result: %w", err)
		}
	}
	return updates, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	body := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	return c.postJSON(ctx, "sendMessage", body)
}

func (c *Client) SendHTMLMessage(ctx context.Context, chatID int64, text string) error {
	body := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}
	return c.postJSON(ctx, "sendMessage", body)
}

func (c *Client) SendDocument(ctx context.Context, chatID int64, path string, caption string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", fmt.Sprintf("%d", chatID))
	if caption != "" {
		_ = w.WriteField("caption", caption)
	}
	part, err := w.CreateFormFile("document", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/sendDocument", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram sendDocument transport: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := readBodyLimited(resp.Body, 1<<20)
	if err != nil {
		return fmt.Errorf("telegram sendDocument read body: %w", err)
	}
	env, err := decodeEnvelope(raw)
	if err != nil {
		return &APIError{
			Method:      "sendDocument",
			HTTPStatus:  resp.StatusCode,
			Description: truncateForLog(string(raw), 200),
		}
	}
	if ae := apiErrorFromEnvelope("sendDocument", resp.StatusCode, env); ae != nil {
		return ae
	}
	if resp.StatusCode/100 != 2 {
		return &APIError{
			Method:      "sendDocument",
			HTTPStatus:  resp.StatusCode,
			Description: truncateForLog(string(raw), 200),
		}
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, method string, body map[string]interface{}) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/"+method, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s transport: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := readBodyLimited(resp.Body, 1<<20)
	if err != nil {
		return fmt.Errorf("telegram %s read body: %w", method, err)
	}
	env, err := decodeEnvelope(respBody)
	if err != nil {
		return &APIError{
			Method:      method,
			HTTPStatus:  resp.StatusCode,
			Description: truncateForLog(string(respBody), 200),
		}
	}
	if ae := apiErrorFromEnvelope(method, resp.StatusCode, env); ae != nil {
		return ae
	}
	if resp.StatusCode/100 != 2 {
		return &APIError{
			Method:      method,
			HTTPStatus:  resp.StatusCode,
			Description: truncateForLog(string(respBody), 200),
		}
	}
	return nil
}

// RetryAfterSec returns Telegram retry_after when error_code is 429.
func RetryAfterSec(err error) int {
	ae, ok := AsAPIError(err)
	if !ok || ae.ErrorCode != 429 {
		return 0
	}
	// Description is often "Too Many Requests: retry after 12"
	lower := strings.ToLower(ae.Description)
	const needle = "retry after "
	if i := strings.Index(lower, needle); i >= 0 {
		rest := strings.TrimSpace(lower[i+len(needle):])
		var sec int
		for _, ch := range rest {
			if ch < '0' || ch > '9' {
				break
			}
			sec = sec*10 + int(ch-'0')
		}
		if sec > 0 {
			return sec
		}
	}
	return 5
}
