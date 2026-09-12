package jobboard

import "testing"

func TestIsJobboardURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url  string
		kind string
	}{
		{"https://jobs.dou.ua/companies/tapok/vacancies/118950/", KindVacancy},
		{"https://jobs.dou.ua/companies/omnia-media/", KindCompany},
		{"https://djinni.co/jobs/796247-dsp-media-buying-team-lead/", KindVacancy},
		{"https://djinni.co/jobs/company-tapok/", KindCompany},
		{"https://djinni.co/q/0fbc619b46/", KindProfile},
		{"https://jobs.dou.ua/vacancies/?category=Marketing", KindUnknown},
		{"https://djinni.co/jobs/?company=tap-ok", KindUnknown},
	}
	for _, tc := range cases {
		if got := Kind(tc.url); got != tc.kind {
			t.Errorf("Kind(%q)=%q want %q", tc.url, got, tc.kind)
		}
		if want := tc.kind != KindUnknown; IsJobboardURL(tc.url) != want {
			t.Errorf("IsJobboardURL(%q)=%v want %v", tc.url, !want, want)
		}
	}
}

func TestCompanySlug(t *testing.T) {
	t.Parallel()
	if slug := CompanySlug("https://jobs.dou.ua/companies/tapok/vacancies/118950/"); slug != "tapok" {
		t.Fatalf("slug=%q", slug)
	}
	if slug := CompanySlug("https://djinni.co/jobs/company-tapok/"); slug != "tapok" {
		t.Fatalf("slug=%q", slug)
	}
}
