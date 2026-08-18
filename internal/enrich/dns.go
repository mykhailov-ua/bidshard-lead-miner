package enrich

import (
	"context"
	"net"
	"strings"

	"github.com/bidshard/parser/internal/scoring"
)

var trackerSubdomains = []string{"track", "click", "go", "trk", "redirect", "tracker", "clicks"}

type DNSLookup struct {
	resolver *net.Resolver
}

func NewDNSLookup() *DNSLookup {
	return &DNSLookup{resolver: net.DefaultResolver}
}

func (d *DNSLookup) TrackerFingerprints(ctx context.Context, domain string) []string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || d == nil {
		return nil
	}

	seen := map[string]struct{}{}
	var out []string
	for _, sub := range trackerSubdomains {
		host := sub + "." + domain
		cname, err := d.resolver.LookupCNAME(ctx, host)
		if err != nil || cname == "" || cname == host+"." {
			continue
		}
		cname = strings.TrimSuffix(strings.ToLower(cname), ".")
		for _, hit := range scoring.DetectCompetitorStack(cname) {
			if _, ok := seen[hit]; ok {
				continue
			}
			seen[hit] = struct{}{}
			out = append(out, hit)
		}
		if len(out) == 0 {
			out = append(out, "cname:"+cname)
		}
	}
	return out
}
