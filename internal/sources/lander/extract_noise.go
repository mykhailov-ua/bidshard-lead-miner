package lander

import (
	"strings"
	"unicode"
)

// skipExtractString drops CDN URLs, asset paths, base64 blobs, and other non-copy
// literals from Next/RSC JSON flattening so keyword prescan does not inflate on noise.
func skipExtractString(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if len(s) <= 2 {
		return true
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//") {
		return true
	}
	if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "blob:") {
		return true
	}
	if strings.HasPrefix(s, "/_next/") || strings.HasPrefix(s, "/static/chunks/") {
		return true
	}
	if isAssetPath(s) {
		return true
	}
	if hasAssetExtension(lower) {
		return true
	}
	if isLikelyBase64Blob(s) {
		return true
	}
	if isWebpackChunkID(s) {
		return true
	}
	if isMIMEType(s) {
		return true
	}
	return false
}

func isAssetPath(s string) bool {
	if !strings.HasPrefix(s, "/") || strings.Contains(s, " ") {
		return false
	}
	lower := strings.ToLower(s)
	for _, ext := range []string{".js", ".css", ".map", ".woff", ".woff2", ".ttf", ".eot", ".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico", ".avif"} {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	return false
}

func hasAssetExtension(s string) bool {
	for _, ext := range []string{".woff2", ".woff", ".ttf", ".eot", ".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico", ".avif", ".css.map", ".js.map"} {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

func isLikelyBase64Blob(s string) bool {
	// Short strings are unlikely to be embedded font or chunk payloads.
	if len(s) < 80 {
		return false
	}
	if strings.Contains(s, " ") {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func isWebpackChunkID(s string) bool {
	// React/webpack content hashes are 12-64 hex chars without spaces.
	if strings.Contains(s, " ") || len(s) < 12 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func isMIMEType(s string) bool {
	if strings.Contains(s, " ") {
		return false
	}
	idx := strings.Index(s, "/")
	if idx <= 0 || idx >= len(s)-1 {
		return false
	}
	prefix := s[:idx]
	if !unicode.IsLetter(rune(prefix[0])) {
		return false
	}
	for _, r := range prefix[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '+' && r != '-' && r != '.' {
			return false
		}
	}
	suffix := s[idx+1:]
	for _, r := range suffix {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '+' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}
