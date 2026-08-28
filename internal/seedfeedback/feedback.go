package seedfeedback

import (
	"strings"
	"sync"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/sources/tgweb"
	"github.com/bidshard/parser/internal/validate"
)

// Config controls accepted-lead -> source_registry feedback.
type Config struct {
	Enabled      bool
	RegistryPath string
	MinHeatTier  string
}

// AcceptInput is one accepted lead on the hot path.
type AcceptInput struct {
	Lead         model.Lead
	Snippet      string
	Contacts     []extract.Contact
	EnrichDomain string
	Username     string
	Title        string
	ForumUserID  string
}

// Recorder enqueues crawl domains from accepted leads (file-backed registry).
type Recorder struct {
	cfg Config
	mu  sync.Mutex
}

func NewRecorder(cfg Config) *Recorder {
	if !cfg.Enabled || strings.TrimSpace(cfg.RegistryPath) == "" {
		return nil
	}
	if cfg.MinHeatTier == "" {
		cfg.MinHeatTier = entity.HeatTierHot
	}
	return &Recorder{cfg: cfg}
}

// RecordAccepted upserts domains into source_registry when gates pass.
func (r *Recorder) RecordAccepted(in AcceptInput) int {
	if r == nil {
		return 0
	}
	hotPath, forumMention := r.enqueuePaths(in)
	if !hotPath && !forumMention {
		return 0
	}

	allDomains := domainsFromAccept(in)
	snippetDomains := snippetDomainSet(in.Snippet)
	if len(allDomains) == 0 {
		return 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	added := 0
	for _, domain := range allDomains {
		discoveredBy := ""
		switch {
		case hotPath:
			discoveredBy = "accepted_lead"
		case forumMention && snippetDomains[domain]:
			discoveredBy = "forum_crossmention"
		default:
			continue
		}
		ok, err := sourceregistry.Upsert(r.cfg.RegistryPath, sourceregistry.Entry{
			Domain:       domain,
			Types:        append([]string(nil), sourceregistry.CascadeTypes...),
			DiscoveredBy: discoveredBy,
			Source:       in.Lead.Source,
			Channel:      in.Lead.Source,
		})
		if err != nil {
			continue
		}
		if ok {
			added++
		}
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("accepted_lead", added)
	}
	return added
}

func (r *Recorder) enqueuePaths(in AcceptInput) (hotPath bool, forumMention bool) {
	minRank := entity.HeatTierRank(r.cfg.MinHeatTier)
	if entity.HeatTierRank(in.Lead.HeatTier) >= minRank {
		hotPath = true
	}
	if scoring.MeetsMinPriority(scoring.Priority(in.Lead.Priority), scoring.PriorityHigh) {
		hotPath = true
	}
	family := entity.SourceFamily(in.Lead.Source)
	forumMention = family == "forum" || family == "warrior"
	return hotPath, forumMention
}

func domainsFromAccept(in AcceptInput) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(domain string) {
		domain = entity.NormalizeDomain(domain)
		if domain == "" || !extract.IsValidWebDomain(domain) {
			return
		}
		if validate.IsBlacklistedDomain(domain) || geo.IsBlockedTLD(domain) {
			return
		}
		if _, ok := seen[domain]; ok {
			return
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}

	if d := tgweb.SiteDomainFromSource(in.Lead.Source); d != "" {
		add(d)
	}
	if d := entity.SupplyDomainFromSource(in.Lead.Source); d != "" {
		add(d)
	}
	if in.EnrichDomain != "" {
		add(in.EnrichDomain)
	}

	resolve := entity.EnrichForumIdentity(
		entity.ResolveInputFromLead(in.Lead, in.Contacts),
		in.Username,
		in.Title,
		in.ForumUserID,
	)
	for _, key := range entity.ResolveKeys(resolve) {
		if key.Kind == entity.KindDomain {
			add(key.Value)
		}
	}
	if filter.IsTgWebSource(in.Lead.Source) {
		return out
	}
	for _, d := range extract.WebDomains(in.Snippet) {
		add(d)
	}
	return out
}

func snippetDomainSet(snippet string) map[string]bool {
	out := map[string]bool{}
	for _, d := range extract.WebDomains(snippet) {
		d = entity.NormalizeDomain(d)
		if d != "" {
			out[d] = true
		}
	}
	return out
}
