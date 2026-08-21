package lander

import (
	"net/url"
	"strings"
)

// ShouldFetchRSC reports whether an App Router page likely has more data via RSC flight HTTP.
func ShouldFetchRSC(html string) bool {
	if strings.Contains(html, "__NEXT_DATA__") {
		// Pages Router already embeds JSON; skip redundant RSC round-trip.
		return false
	}
	if HasNextFlightPayload(html) {
		return true
	}
	if strings.Contains(html, "/_next/static/chunks/app/") {
		return true
	}
	if strings.Contains(html, "id=\"__next\"") || strings.Contains(html, "id='__next'") {
		return true
	}
	return false
}

// ExtractFlightWireText parses raw React Flight wire payloads (RSC HTTP responses).
func ExtractFlightWireText(payload string) string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return ""
	}

	var parts []string
	for line := range strings.SplitSeq(payload, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if chunk := flattenFlightChunk(line); chunk != "" {
			parts = append(parts, chunk)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return collectRSCStrings(payload)
}

func wrapRSCAsFlightScripts(payload string) string {
	// Synthesize __next_f.push lines so HasNextFlightPayload / DiagnoseExtract can consume RSC wire.
	var scripts []string
	for line := range strings.SplitSeq(strings.TrimSpace(payload), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		escaped := strings.ReplaceAll(line, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		scripts = append(scripts, `self.__next_f.push([1,"`+escaped+`"])`)
	}
	return strings.Join(scripts, "\n")
}

func pagePath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return "/"
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}

func resolveFetchURL(baseURL, rawURL string) string {
	if baseURL == "" {
		return rawURL
	}
	// Tracker landers: preserve /offer/ and /preview paths; map bare seeds to tracker base root.
	if idx := strings.Index(rawURL, "/offer/"); idx >= 0 {
		return baseURL + rawURL[idx:]
	}
	if idx := strings.Index(rawURL, "/preview"); idx >= 0 {
		return baseURL + rawURL[idx:]
	}
	return baseURL + "/"
}
