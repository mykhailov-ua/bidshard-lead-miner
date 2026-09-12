package jobboard

import (
	"strings"
	"testing"
)

const douVacancyFixture = `<html><head>
<meta property="og:title" content="Media Buyer в TapOK">
<meta name="description" content="Keitaro Binom media buyer gambling">
</head><body>
<div class="b-compinfo"><a href="/companies/tapok/">TapOK</a></div>
<h1 class="g-h2">Media Buyer</h1>
<div class="b-typo vacancy-section">Experience with Keitaro and Binom trackers.</div>
<div class="b-typo vacancy-section">Budget from $500/day in gambling vertical.</div>
</body></html>`

const djinniVacancyFixture = `<html><head>
<meta property="og:title" content="DSP Media Buying Team Lead в TapOk">
</head><body>
<a href="/jobs/company-tapok/">TapOk</a>
<div class="job-post__description">Search arbitrage team with Binom and Clickflare. Budget $120k/month.</div>
</body></html>`

const djinniProfileFixture = `<html><head><meta name="description" content="Senior Media Buyer Unlim Click"></head><body>
<section id="chapter-experience"><p class="profile">Team Lead at Unlim Click. Keitaro, gambling, $145k spend.</p></section>
</body></html>`

func TestParseHTMLDouVacancy(t *testing.T) {
	t.Parallel()
	page, ok := ParseHTML("https://jobs.dou.ua/companies/tapok/vacancies/118950/", douVacancyFixture)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if page.Company != "TapOK" {
		t.Fatalf("company=%q", page.Company)
	}
	if !containsAll(page.Body, "Keitaro", "gambling") {
		t.Fatalf("body=%q", page.Body)
	}
}

func TestParseHTMLDjinniVacancy(t *testing.T) {
	t.Parallel()
	page, ok := ParseHTML("https://djinni.co/jobs/796247-dsp-media-buying-team-lead/", djinniVacancyFixture)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if page.Company != "TapOk" {
		t.Fatalf("company=%q", page.Company)
	}
	if !containsAll(page.Body, "Binom", "Search arbitrage") {
		t.Fatalf("body=%q", page.Body)
	}
}

func TestParseHTMLDjinniProfile(t *testing.T) {
	t.Parallel()
	page, ok := ParseHTML("https://djinni.co/q/0fbc619b46/", djinniProfileFixture)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if page.Kind != KindProfile {
		t.Fatalf("kind=%q", page.Kind)
	}
	if !containsAll(page.Body, "Unlim Click", "Keitaro") {
		t.Fatalf("body=%q", page.Body)
	}
}

func containsAll(text string, parts ...string) bool {
	lower := strings.ToLower(text)
	for _, p := range parts {
		if !strings.Contains(lower, strings.ToLower(p)) {
			return false
		}
	}
	return true
}
