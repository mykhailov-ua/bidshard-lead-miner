package lander

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestExtractPageTextFromNextData(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/lander/next_page.html")
	if err != nil {
		t.Fatal(err)
	}

	text, err := ExtractPageText(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(text), "voluum alternative") {
		t.Fatalf("missing keyword in %q", text)
	}
	if !strings.Contains(text, "postback failing") {
		t.Fatalf("missing pain text in %q", text)
	}
}

func TestHasNextFlightPayload(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/lander/next_app_router.html")
	if err != nil {
		t.Fatal(err)
	}
	if !HasNextFlightPayload(string(html)) {
		t.Fatal("expected flight payload detection")
	}
}

func TestExtractPageTextFromNextFlight(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/lander/next_app_router.html")
	if err != nil {
		t.Fatal(err)
	}

	text, err := ExtractPageText(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(text), "voluum alternative") {
		t.Fatalf("missing keyword in %q", text)
	}
	if !strings.Contains(text, "postback failing") {
		t.Fatalf("missing pain text in %q", text)
	}
}

func TestHeadlessDisabledByDefault(t *testing.T) {
	t.Parallel()

	_, err := DisabledHeadless{}.Fetch(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected headless disabled error")
	}
}
