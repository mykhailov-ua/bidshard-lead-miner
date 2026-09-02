package pipeline

import (
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/parser/internal/model"
)

type RoundState struct {
	RawTotal                  atomic.Int64
	Accepted                  atomic.Int64
	RejectedGeo               atomic.Int64
	HardRejected              atomic.Int64
	Dedup                     atomic.Int64
	Dropped                   atomic.Int64
	High                      atomic.Int64
	Medium                    atomic.Int64
	Low                       atomic.Int64
	RejectedBlacklist         atomic.Int64
	RejectedIntelOnly         atomic.Int64
	RejectedLanderNoBuyer     atomic.Int64
	RejectedGitHubVendor      atomic.Int64
	RejectedTelegramSpam      atomic.Int64
	RejectedLowPriority       atomic.Int64
	RejectedICP               atomic.Int64
	RejectedIntent            atomic.Int64
	RejectedLang              atomic.Int64
	RejectedContext           atomic.Int64
	RejectedContact           atomic.Int64
	RejectedNoContacts        atomic.Int64
	RejectedEmailNoContext    atomic.Int64
	RejectedRoleEmail         atomic.Int64
	RejectedEmptyHash         atomic.Int64
	RejectedMX                atomic.Int64
	SourcesOK                 atomic.Int64
	SourcesFail               atomic.Int64

	mu    sync.Mutex
	leads []model.Lead
	wg    sync.WaitGroup
}

func (s *RoundState) Wait() {
	s.wg.Wait()
}

func (s *RoundState) TrackTask() {
	s.wg.Add(1)
}

func (s *RoundState) FinishTask() {
	s.wg.Done()
}

func (s *RoundState) AddLead(lead model.Lead) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leads = append(s.leads, lead)
}

func (s *RoundState) Leads() []model.Lead {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Lead, len(s.leads))
	copy(out, s.leads)
	return out
}

type RoundStats struct {
	RoundID               string
	Duration              time.Duration
	SourcesOK             int
	SourcesFail           int
	RawTotal              int
	Accepted              int
	RejectedGeo           int
	HardRejected          int
	Dedup                 int
	Dropped               int
	High                  int
	Medium                int
	Low                   int
	RejectedBlacklist     int
	RejectedIntelOnly     int
	RejectedLanderNoBuyer int
	RejectedGitHubVendor  int
	RejectedTelegramSpam  int
	RejectedLowPriority   int
	RejectedICP           int
	RejectedIntent        int
	RejectedLang          int
	RejectedContext       int
	RejectedContact       int
	RejectedNoContacts    int
	RejectedEmailNoContext int
	RejectedRoleEmail     int
	RejectedEmptyHash     int
	RejectedMX            int
	Leads                 []model.Lead
}

func (s *RoundState) Snapshot(roundID string, duration time.Duration) RoundStats {
	return RoundStats{
		RoundID:               roundID,
		Duration:              duration,
		SourcesOK:             int(s.SourcesOK.Load()),
		SourcesFail:           int(s.SourcesFail.Load()),
		RawTotal:              int(s.RawTotal.Load()),
		Accepted:              int(s.Accepted.Load()),
		RejectedGeo:           int(s.RejectedGeo.Load()),
		HardRejected:          int(s.HardRejected.Load()),
		Dedup:                 int(s.Dedup.Load()),
		Dropped:               int(s.Dropped.Load()),
		High:                  int(s.High.Load()),
		Medium:                int(s.Medium.Load()),
		Low:                   int(s.Low.Load()),
		RejectedBlacklist:     int(s.RejectedBlacklist.Load()),
		RejectedIntelOnly:     int(s.RejectedIntelOnly.Load()),
		RejectedLanderNoBuyer: int(s.RejectedLanderNoBuyer.Load()),
		RejectedGitHubVendor:  int(s.RejectedGitHubVendor.Load()),
		RejectedTelegramSpam:  int(s.RejectedTelegramSpam.Load()),
		RejectedLowPriority:   int(s.RejectedLowPriority.Load()),
		RejectedICP:            int(s.RejectedICP.Load()),
		RejectedIntent:         int(s.RejectedIntent.Load()),
		RejectedLang:           int(s.RejectedLang.Load()),
		RejectedContext:        int(s.RejectedContext.Load()),
		RejectedContact:        int(s.RejectedContact.Load()),
		RejectedNoContacts:     int(s.RejectedNoContacts.Load()),
		RejectedEmailNoContext: int(s.RejectedEmailNoContext.Load()),
		RejectedRoleEmail:      int(s.RejectedRoleEmail.Load()),
		RejectedEmptyHash:      int(s.RejectedEmptyHash.Load()),
		RejectedMX:             int(s.RejectedMX.Load()),
		Leads:                  s.Leads(),
	}
}

// TopRejectReasons returns up to n "reason:count" strings sorted by count desc.
func TopRejectReasons(s RoundStats, n int) []string {
	buckets := map[string]int{
		"geo":                      s.RejectedGeo,
		"hard_reject":              s.HardRejected,
		"dedup":                    s.Dedup,
		"blacklist":                s.RejectedBlacklist,
		"intel_only":               s.RejectedIntelOnly,
		"lander_no_buyer_signal":   s.RejectedLanderNoBuyer,
		"github_vendor":            s.RejectedGitHubVendor,
		"telegram_spam":            s.RejectedTelegramSpam,
		"low_priority":             s.RejectedLowPriority,
		"icp":                      s.RejectedICP,
		"intent":                   s.RejectedIntent,
		"lang":                     s.RejectedLang,
		"context":                  s.RejectedContext,
		"contact":                  s.RejectedContact,
		"no_contacts":              s.RejectedNoContacts,
		"email_no_context":         s.RejectedEmailNoContext,
		"role_email":               s.RejectedRoleEmail,
		"empty_hash":               s.RejectedEmptyHash,
		"mx":                       s.RejectedMX,
		"dropped":                  s.Dropped,
	}
	return topRejectReasons(buckets, n)
}

func topRejectReasons(reasons map[string]int, n int) []string {
	type pair struct {
		reason string
		count  int
	}
	var pairs []pair
	for reason, count := range reasons {
		if count <= 0 {
			continue
		}
		pairs = append(pairs, pair{reason, count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].reason < pairs[j].reason
		}
		return pairs[i].count > pairs[j].count
	})
	if n > 0 && len(pairs) > n {
		pairs = pairs[:n]
	}
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.reason+":"+strconv.Itoa(p.count))
	}
	return out
}
