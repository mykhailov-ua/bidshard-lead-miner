package lander

import (
	"strings"
	"testing"
)

func TestSkipExtractString(t *testing.T) {
	t.Parallel()

	skip := []string{
		"",
		"  ",
		"https://framerusercontent.com/assets/font.woff2",
		"//cdn.example.com/chunk.js",
		"/_next/static/chunks/app/page.js",
		"/static/chunks/webpack-abc123.js",
		"image/png",
		"application/javascript",
		"deadbeefcafebabe",
	}
	for _, s := range skip {
		if !skipExtractString(s) {
			t.Fatalf("expected skip for %q", s)
		}
	}

	keep := []string{
		"voluum alternative",
		"postback failing on tracker",
		"Affiliate program for media buyers",
		"partnerships@topxpartners.com",
	}
	for _, s := range keep {
		if skipExtractString(s) {
			t.Fatalf("expected keep for %q", s)
		}
	}
}

func TestFlattenJSONSkipsNoise(t *testing.T) {
	t.Parallel()

	raw := `{
		"title": "voluum alternative",
		"asset": "https://cdn.example.com/app.js",
		"chunk": "/_next/static/chunks/main.js",
		"pain": "postback failing",
		"blob": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="
	}`
	text := flattenJSON(raw)
	if !strings.Contains(text, "voluum alternative") || !strings.Contains(text, "postback failing") {
		t.Fatalf("missing copy in %q", text)
	}
	for _, noise := range []string{"cdn.example.com", "/_next/static", "YWJjZGVm"} {
		if strings.Contains(text, noise) {
			t.Fatalf("noise %q in %q", noise, text)
		}
	}
}

func TestCollectRSCStringsSkipsURLs(t *testing.T) {
	t.Parallel()

	raw := `"voluum alternative" "https://example.com/font.woff2" "postback failing" "/_next/static/chunks/app.js"`
	text := collectRSCStrings(raw)
	if !strings.Contains(text, "voluum alternative") || !strings.Contains(text, "postback failing") {
		t.Fatalf("missing copy in %q", text)
	}
	for _, noise := range []string{"example.com", "/_next/static"} {
		if strings.Contains(text, noise) {
			t.Fatalf("noise %q in %q", noise, text)
		}
	}
}
