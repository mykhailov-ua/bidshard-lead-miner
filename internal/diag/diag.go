package diag

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultPreview = 500
	HTMLPreview    = 800
)

var scriptStripRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)

// Preview returns a single-line truncated string for logs.
func Preview(s string, max int) string {
	if max <= 0 {
		max = DefaultPreview
	}
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// PreviewHTML logs HTML size and a stripped text preview (scripts removed).
func PreviewHTML(html string, max int) string {
	if max <= 0 {
		max = HTMLPreview
	}
	stripped := scriptStripRe.ReplaceAllString(html, " ")
	stripped = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(stripped, " ")
	return Preview(stripped, max)
}

// MaskContact masks contact hints for logs (email/telegram).
func MaskContact(contact string) string {
	contact = strings.TrimSpace(contact)
	if contact == "" {
		return ""
	}
	if strings.Contains(contact, "@") {
		parts := strings.SplitN(contact, "@", 2)
		if len(parts[0]) == 0 {
			return "***@" + parts[1]
		}
		return parts[0][:1] + "***@" + parts[1]
	}
	if len(contact) <= 3 {
		return "***"
	}
	return contact[:1] + "***"
}

// ByteSize formats byte counts for structured logs.
func ByteSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
}
