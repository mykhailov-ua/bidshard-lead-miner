package enrich

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/httpclient"
)

type EmailLookup struct {
	client      *http.Client
	smtpVerify  bool
	smtpTimeout time.Duration
}

func NewEmailLookup(client *http.Client, smtpVerify bool, smtpTimeout time.Duration) *EmailLookup {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if smtpTimeout <= 0 {
		smtpTimeout = 8 * time.Second
	}
	return &EmailLookup{client: client, smtpVerify: smtpVerify, smtpTimeout: smtpTimeout}
}

type EmailInfo struct {
	DisplayName string
	Gravatar    string
	HasGravatar bool
	SMTPValid   bool
	SMTPChecked bool
}

func ParseDisplayName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(addr.Name)
}

func (e *EmailLookup) Lookup(ctx context.Context, email, displayHint string) (EmailInfo, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return EmailInfo{}, nil
	}
	info := EmailInfo{DisplayName: ParseDisplayName(displayHint)}
	if info.DisplayName == "" {
		info.DisplayName = ParseDisplayName(email)
	}

	name, ok, err := e.gravatarName(ctx, email)
	if err == nil {
		info.HasGravatar = ok
		info.Gravatar = name
		if info.DisplayName == "" && name != "" {
			info.DisplayName = name
		}
	}

	if e.smtpVerify {
		valid, err := verifySMTP(ctx, email, e.smtpTimeout)
		info.SMTPChecked = err == nil
		info.SMTPValid = valid
	}
	return info, nil
}

func (e *EmailLookup) gravatarName(ctx context.Context, email string) (string, bool, error) {
	if e == nil || e.client == nil {
		return "", false, nil
	}
	sum := md5.Sum([]byte(strings.TrimSpace(strings.ToLower(email))))
	url := fmt.Sprintf("https://www.gravatar.com/%s.json", hex.EncodeToString(sum[:]))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return "", false, err
	}
	body, err := httpclient.ReadResponseBody(resp, 64<<10)
	if err != nil {
		return "", false, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("gravatar http %d", resp.StatusCode)
	}
	var parsed []struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", false, err
	}
	if len(parsed) == 0 || parsed[0].DisplayName == "" {
		return "", true, nil
	}
	return parsed[0].DisplayName, true, nil
}

func verifySMTP(ctx context.Context, email string, timeout time.Duration) (bool, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	mxRecords, err := net.DefaultResolver.LookupMX(lookupCtx, parts[1])
	if err != nil || len(mxRecords) == 0 {
		return false, err
	}
	host := strings.TrimSuffix(mxRecords[0].Host, ".")
	addr := host + ":25"

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(lookupCtx, "tcp", addr)
	if err != nil {
		return false, nil
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return false, nil
	}
	defer func() { _ = client.Close() }()

	_ = client.Hello("bidshard-parser.local")
	_ = client.Mail("probe@bidshard-parser.local")
	if err := client.Rcpt(email); err != nil {
		return false, nil
	}
	return true, nil
}
