package lander

import (
	"os"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestExtractStaticLandingTextFooter(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/lander/affiliate_tailwind.html")
	if err != nil {
		t.Fatal(err)
	}
	text := ExtractStaticLandingText(string(html))
	if !strings.Contains(text, "partnerships@topxpartners.com") {
		t.Fatalf("missing footer email in %q", text)
	}
	if !strings.Contains(text, "affiliates@topxpartners.com") {
		t.Fatalf("missing mailto email in %q", text)
	}
	if !strings.Contains(strings.ToLower(text), "skype") {
		t.Fatalf("missing skype hint in %q", text)
	}
}

func TestTextForContactExtractStaticOnly(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/lander/affiliate_tailwind.html")
	if err != nil {
		t.Fatal(err)
	}
	text, method := TextForContactExtract(string(html))
	if method != "static_html" && !strings.Contains(method, "static") {
		t.Fatalf("method=%q want static", method)
	}
	contacts := extract.Extract(text)
	if len(contacts.Contacts) == 0 {
		t.Fatalf("expected contacts from static landing, text=%q", text)
	}
	foundAffiliates := false
	for _, c := range contacts.Contacts {
		if c.Type == "email" && strings.Contains(c.Value, "affiliates@topxpartners.com") {
			foundAffiliates = true
		}
	}
	if !foundAffiliates {
		t.Fatalf("contacts=%v", contacts.Contacts)
	}
}

func TestNetworkLandingPathsIncludesAffiliateProgram(t *testing.T) {
	t.Parallel()

	paths := NetworkLandingPaths()
	want := map[string]bool{
		"/affiliate-program": true,
		"/become-a-partner":  true,
	}
	for _, p := range paths {
		delete(want, p)
	}
	if len(want) != 0 {
		t.Fatalf("missing paths %v in %v", want, paths)
	}
}

func TestNetworkLandingPathsPrioritizesAboutBeforeAffiliate(t *testing.T) {
	t.Parallel()

	paths := NetworkLandingPaths()
	aboutIdx := indexOf(paths, "/about")
	contactIdx := indexOf(paths, "/contact")
	affiliateIdx := indexOf(paths, "/affiliate-program")
	if aboutIdx < 0 || contactIdx < 0 || affiliateIdx < 0 {
		t.Fatalf("missing expected paths in %v", paths)
	}
	if aboutIdx >= contactIdx {
		t.Fatalf("about should precede contact for SPA sites: %v", paths)
	}
	if aboutIdx >= affiliateIdx {
		t.Fatalf("about should precede affiliate paths: %v", paths)
	}
}

func TestExtractStaticLandingTextSkipsBodyWhenContactRegionFound(t *testing.T) {
	t.Parallel()

	html := `<html><body>
	<main><p>UNIQUE_MAIN_COPY_SHOULD_NOT_APPEAR_IN_RAW</p></main>
	<footer>partnerships@example.com</footer>
	</body></html>`
	text := ExtractStaticLandingText(html)
	if !strings.Contains(text, "partnerships@example.com") {
		t.Fatalf("missing footer email in %q", text)
	}
	if strings.Contains(text, "UNIQUE_MAIN_COPY_SHOULD_NOT_APPEAR_IN_RAW") {
		t.Fatalf("body copy should be skipped when footer contact found: %q", text)
	}
}

func TestExtractVisibleBodyTextStripsInlineStyle(t *testing.T) {
	t.Parallel()

	html := `<div style="@media screen { color: red; }">visible copy only</div>`
	text := extractVisibleBodyText(html, 50000)
	if strings.Contains(text, "@media") {
		t.Fatalf("inline style leaked into text: %q", text)
	}
	if !strings.Contains(text, "visible copy only") {
		t.Fatalf("missing visible copy in %q", text)
	}
}

func indexOf(items []string, want string) int {
	for i, item := range items {
		if item == want {
			return i
		}
	}
	return -1
}
