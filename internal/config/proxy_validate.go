package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateProxyURLs rejects PARSER_PROXY_LIST entries that dial as tcp :0 or empty host.
func ValidateProxyURLs(raw []string) error {
	if len(raw) == 0 {
		return nil
	}
	var msgs []string
	for _, entry := range raw {
		if err := validateProxyURL(entry); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}

func validateProxyURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty proxy URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid proxy URL %q: %w", redactProxyURL(raw), err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("proxy URL %q: scheme must be http or https", redactProxyURL(raw))
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("proxy URL %q: host is empty (check PARSER_PROXY_LIST template vars)", redactProxyURL(raw))
	}
	if port := u.Port(); port == "0" {
		return fmt.Errorf("proxy URL %q: port 0 is invalid", redactProxyURL(raw))
	}
	return nil
}

func redactProxyURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Redacted()
}
