package telegrambot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// APIError is a parsed Telegram Bot API error (HTTP or ok=false body).
type APIError struct {
	Method      string
	HTTPStatus  int
	ErrorCode   int
	Description string
}

func (e *APIError) Error() string {
	return e.String()
}

func (e *APIError) String() string {
	if e == nil {
		return "telegram api error"
	}
	parts := []string{e.Method}
	if e.ErrorCode != 0 {
		parts = append(parts, fmt.Sprintf("code=%d", e.ErrorCode))
	}
	if e.HTTPStatus != 0 {
		parts = append(parts, fmt.Sprintf("http=%d", e.HTTPStatus))
	}
	desc := strings.TrimSpace(e.Description)
	if desc != "" {
		parts = append(parts, desc)
	}
	if hint := e.Hint(); hint != "" {
		parts = append(parts, "("+hint+")")
	}
	return "telegram " + strings.Join(parts, " ")
}

// Hint gives ops-friendly cause when the bot does not respond.
func (e *APIError) Hint() string {
	if e == nil {
		return ""
	}
	switch e.ErrorCode {
	case 401:
		return "invalid bot token; check CRM_TELEGRAM_BOT_TOKEN / TELEGRAM_ALERT_BOT_TOKEN"
	case 403:
		return "bot blocked, kicked, or not a member of the chat"
	case 400:
		if strings.Contains(strings.ToLower(e.Description), "chat not found") {
			return "wrong chat_id or bot never added to the group"
		}
		if strings.Contains(strings.ToLower(e.Description), "parse") {
			return "HTML parse error in message text"
		}
		return "bad request; check chat_id and payload"
	case 409:
		return "conflict: another getUpdates long poll is running for this bot"
	case 429:
		return "rate limited; reduce send rate or wait retry_after"
	case 0:
		if e.HTTPStatus == 502 || e.HTTPStatus == 503 {
			return "telegram api temporarily unavailable"
		}
	}
	return ""
}

// LogAttrs returns slog key-value pairs for structured logging.
func (e *APIError) LogAttrs() []any {
	if e == nil {
		return []any{"error", "telegram api"}
	}
	attrs := []any{
		"telegram_method", e.Method,
		"telegram_error_code", e.ErrorCode,
		"telegram_description", redactSecrets(e.Description),
	}
	if e.HTTPStatus != 0 {
		attrs = append(attrs, "http_status", e.HTTPStatus)
	}
	if hint := e.Hint(); hint != "" {
		attrs = append(attrs, "telegram_hint", hint)
	}
	return attrs
}

func AsAPIError(err error) (*APIError, bool) {
	if err == nil {
		return nil, false
	}
	if ae, ok := err.(*APIError); ok {
		return ae, true
	}
	return nil, false
}

type telegramEnvelope struct {
	OK          bool            `json:"ok"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func decodeEnvelope(body []byte) (telegramEnvelope, error) {
	var env telegramEnvelope
	if len(body) == 0 {
		return env, errors.New("empty telegram api response")
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return env, fmt.Errorf("telegram response not json: %s", truncateForLog(string(body), 200))
	}
	return env, nil
}

func apiErrorFromEnvelope(method string, httpStatus int, env telegramEnvelope) *APIError {
	if env.OK {
		return nil
	}
	return &APIError{
		Method:      method,
		HTTPStatus:  httpStatus,
		ErrorCode:   env.ErrorCode,
		Description: strings.TrimSpace(env.Description),
	}
}

func readBodyLimited(r io.Reader, max int) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, int64(max)))
}

func redactSecrets(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Never echo bot tokens if they ever appear in upstream error text.
	const marker = "/bot"
	if i := strings.Index(s, marker); i >= 0 {
		return s[:i+len(marker)] + "<redacted>"
	}
	return truncateForLog(s, 500)
}

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
