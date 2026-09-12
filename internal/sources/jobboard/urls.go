package jobboard

import (
	"net/url"
	"strings"
)

const (
	KindVacancy = "vacancy"
	KindCompany = "company"
	KindProfile = "profile"
	KindUnknown = "unknown"
	HostDOU     = "jobs.dou.ua"
	HostDjinni  = "djinni.co"
)

// IsJobboardURL reports crawlable DOU/Djinni vacancy, company, or public profile pages.
func IsJobboardURL(rawURL string) bool {
	return Kind(rawURL) != KindUnknown
}

// Kind classifies a job-board URL.
func Kind(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return KindUnknown
	}
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	path := strings.TrimSuffix(strings.ToLower(u.Path), "/")
	switch host {
	case HostDOU:
		return douKind(path)
	case HostDjinni:
		return djinniKind(path)
	default:
		return KindUnknown
	}
}

func douKind(path string) string {
	if strings.HasPrefix(path, "/companies/") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		// companies/{slug}/vacancies/{id}
		if len(parts) >= 4 && parts[2] == "vacancies" && parts[3] != "" {
			return KindVacancy
		}
		// companies/{slug}
		if len(parts) == 2 && parts[1] != "" && parts[1] != "companies" {
			return KindCompany
		}
	}
	return KindUnknown
}

func djinniKind(path string) string {
	if strings.HasPrefix(path, "/q/") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 2 && parts[1] != "" {
			return KindProfile
		}
	}
	if !strings.HasPrefix(path, "/jobs/") {
		return KindUnknown
	}
	rest := strings.TrimPrefix(path, "/jobs/")
	if rest == "" || strings.HasPrefix(rest, "?") {
		return KindUnknown
	}
	segment := rest
	if idx := strings.Index(segment, "/"); idx >= 0 {
		segment = segment[:idx]
	}
	if segment == "" {
		return KindUnknown
	}
	if strings.HasPrefix(segment, "company-") {
		return KindCompany
	}
	// numeric id prefix: 796247-dsp-media-buying-team-lead
	for i, ch := range segment {
		if ch == '-' {
			if i > 0 {
				return KindVacancy
			}
			break
		}
		if ch < '0' || ch > '9' {
			break
		}
		if i == len(segment)-1 {
			return KindVacancy
		}
	}
	return KindUnknown
}

// CompanySlug extracts employer slug from a job-board URL when present.
func CompanySlug(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	path := strings.Trim(strings.TrimSuffix(u.Path, "/"), "/")
	parts := strings.Split(path, "/")
	switch host {
	case HostDOU:
		if len(parts) >= 2 && parts[0] == "companies" {
			return parts[1]
		}
	case HostDjinni:
		if len(parts) >= 2 && parts[0] == "jobs" && strings.HasPrefix(parts[1], "company-") {
			return strings.TrimPrefix(parts[1], "company-")
		}
		if len(parts) >= 3 && parts[0] == "jobs" && strings.HasPrefix(parts[1], "company-") {
			return strings.TrimPrefix(parts[1], "company-")
		}
	}
	return ""
}

func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Fragment = ""
	return strings.TrimSuffix(u.String(), "/")
}

func HostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(u.Host, "www."))
}
