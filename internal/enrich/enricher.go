package enrich

import (
	"context"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/scoring"
)

type Config struct {
	BlockedCountries []string
	RDAPEnabled      bool
	DNSEnabled       bool
	EmailEnabled     bool
	SMTPVerify       bool
}

type Enricher struct {
	cfg   Config
	rdap  *RDAPLookup
	dns   *DNSLookup
	email *EmailLookup
}

func New(cfg Config, rdap *RDAPLookup, dns *DNSLookup, email *EmailLookup) *Enricher {
	return &Enricher{cfg: cfg, rdap: rdap, dns: dns, email: email}
}

type Input struct {
	Source      string
	Contacts    []extract.Contact
	DisplayHint string
}

type Result struct {
	Domain        string
	RDAPCountry   string
	DomainAgeDays int
	Stack         []string
	DisplayName   string
	GravatarName  string
	HasGravatar   bool
	SMTPValid     bool
	SMTPChecked   bool
	GeoBlocked    bool
	GeoReason     string
}

func (e *Enricher) Enrich(ctx context.Context, in Input) Result {
	if e == nil {
		return Result{}
	}
	domain := TargetDomain(in.Source, in.Contacts)
	out := Result{Domain: domain}

	if e.cfg.DNSEnabled && e.dns != nil && domain != "" {
		out.Stack = append(out.Stack, e.dns.TrackerFingerprints(ctx, domain)...)
	}

	if e.cfg.RDAPEnabled && e.rdap != nil && domain != "" {
		if info, err := e.rdap.Lookup(ctx, domain); err == nil {
			out.RDAPCountry = info.Country
			if !info.CreatedAt.IsZero() {
				out.DomainAgeDays = int(time.Since(info.CreatedAt).Hours() / 24)
			}
			if info.Blocked(e.cfg.BlockedCountries) {
				out.GeoBlocked = true
				out.GeoReason = "rdap country " + info.Country
			}
		}
	}

	email := primaryEmail(in.Contacts)
	if e.cfg.EmailEnabled && e.email != nil && email != "" {
		if info, err := e.email.Lookup(ctx, email, in.DisplayHint); err == nil {
			out.DisplayName = info.DisplayName
			out.GravatarName = info.Gravatar
			out.HasGravatar = info.HasGravatar
			out.SMTPValid = info.SMTPValid
			out.SMTPChecked = info.SMTPChecked
		}
	}

	out.Stack = dedupeStack(out.Stack)
	return out
}

func MergeStack(base, extra []string) []string {
	return dedupeStack(append(append([]string(nil), base...), extra...))
}

func dedupeStack(stack []string) []string {
	if len(stack) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(stack))
	for _, s := range stack {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// CompetitorStackFromResult returns competitor hits for scoring boosts.
func CompetitorStackFromResult(stack []string) []string {
	var out []string
	for _, s := range stack {
		for _, hit := range scoring.DetectCompetitorStack(s) {
			out = append(out, hit)
		}
		if strings.HasPrefix(s, "cname:") {
			out = append(out, s)
		}
	}
	return dedupeStack(out)
}
