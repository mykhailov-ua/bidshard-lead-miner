package lander

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const spa404MinBodyBytes = 400

// SPAShellFingerprint hashes normalized HTML for comparing SPA soft-404 shells across paths.
func SPAShellFingerprint(body string) string {
	norm := normalizeSPAHTML(body)
	if norm == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}

// IsSPA404Shell reports whether a non-OK body looks like an HTML SPA route shell (not a tiny error page).
func IsSPA404Shell(status int, body string) bool {
	if status != 404 {
		return false
	}
	trimmed := strings.TrimSpace(body)
	if len(trimmed) < spa404MinBodyBytes {
		return false
	}
	lower := strings.ToLower(trimmed)
	return strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype")
}

func normalizeSPAHTML(body string) string {
	// Strip volatile script/style blocks before hashing; analytics diffs must not break fingerprint match.
	lower := strings.ToLower(body)
	for _, tag := range []string{"<script", "<style", "<noscript"} {
		for {
			start := strings.Index(lower, tag)
			if start < 0 {
				break
			}
			end := strings.Index(lower[start:], "</")
			if end < 0 {
				break
			}
			closeStart := strings.Index(lower[start+end:], ">")
			if closeStart < 0 {
				break
			}
			closeIdx := start + end + closeStart + 1
			body = body[:start] + body[closeIdx+1:]
			lower = strings.ToLower(body)
		}
	}
	compact := strings.Join(strings.Fields(body), " ")
	if len(compact) > 4096 {
		// Keep fingerprint bounded; shell layout repeats within first 4 KiB of normalized text.
		compact = compact[:4096]
	}
	return compact
}
