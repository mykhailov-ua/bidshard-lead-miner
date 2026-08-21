package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/httpclient"
)

var rdapCountryRe = regexp.MustCompile(`"country"\s*,\s*"text"\s*,\s*"([A-Z]{2})"`)

type RDAPLookup struct {
	client  *http.Client
	baseURL string
}

func NewRDAPLookup(client *http.Client) *RDAPLookup {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &RDAPLookup{client: client, baseURL: "https://rdap.org/domain/"}
}

type RDAPInfo struct {
	Country   string
	CreatedAt time.Time
}

func (r *RDAPLookup) Lookup(ctx context.Context, domain string) (RDAPInfo, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return RDAPInfo{}, nil
	}
	if r == nil {
		return RDAPInfo{}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+domain, nil)
	if err != nil {
		return RDAPInfo{}, err
	}
	req.Header.Set("Accept", "application/rdap+json, application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return RDAPInfo{}, err
	}

	body, err := httpclient.ReadResponseBody(resp, 1<<20)
	if err != nil {
		return RDAPInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RDAPInfo{}, fmt.Errorf("rdap http %d", resp.StatusCode)
	}

	return parseRDAP(body), nil
}

func parseRDAP(body []byte) RDAPInfo {
	var top struct {
		Country string `json:"country"`
		Events  []struct {
			EventAction string `json:"eventAction"`
			EventDate   string `json:"eventDate"`
		} `json:"events"`
	}
	_ = json.Unmarshal(body, &top)

	info := RDAPInfo{Country: strings.ToUpper(strings.TrimSpace(top.Country))}
	if info.Country == "" {
		if m := rdapCountryRe.FindSubmatch(body); len(m) > 1 {
			info.Country = string(m[1])
		}
	}
	for _, e := range top.Events {
		if strings.EqualFold(e.EventAction, "registration") && e.EventDate != "" {
			if t, err := time.Parse(time.RFC3339, e.EventDate); err == nil {
				info.CreatedAt = t.UTC()
				break
			}
		}
	}
	return info
}

func (info RDAPInfo) Blocked(blocked []string) bool {
	return geo.IsBlockedCountry(info.Country, blocked)
}
