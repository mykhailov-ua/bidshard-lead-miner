package discover

import (
	"encoding/json"
	"os"
	"strings"
)

// InfraCluster is weekly Shodan/Censys export row (H6 intel branch, not auto-CRM).
type InfraCluster struct {
	IP           string   `json:"ip"`
	Domains      []string `json:"domains"`
	TrackerHint  string   `json:"tracker_hint"`
	ASN          string   `json:"asn"`
	Country      string   `json:"country"`
	Score        int      `json:"score"`
	SampleDomain string   `json:"sample_domain"`
	Notes        string   `json:"notes"`
}

// InfraClusterFile is data/runtime/infra_clusters.json shape.
type InfraClusterFile struct {
	ExportedAt string         `json:"exported_at"`
	Source     string         `json:"source"`
	Clusters   []InfraCluster `json:"clusters"`
}

const minDomainsPerCluster = 3

// ScoreInfraCluster ranks cluster for manual outreach triage.
func ScoreInfraCluster(c InfraCluster) int {
	score := 0
	if len(c.Domains) >= minDomainsPerCluster {
		score += 20
	}
	if len(c.Domains) >= 10 {
		score += 15
	}
	hint := strings.ToLower(strings.TrimSpace(c.TrackerHint))
	switch {
	case strings.Contains(hint, "keitaro"), strings.Contains(hint, "kclick"):
		score += 25
	case strings.Contains(hint, "binom"), strings.Contains(hint, "voluum"):
		score += 15
	}
	cc := strings.ToUpper(strings.TrimSpace(c.Country))
	if cc != "" && cc != "RU" && cc != "BY" {
		score += 10
	}
	if strings.TrimSpace(c.SampleDomain) != "" {
		score += 5
	}
	return score
}

// LoadInfraClusters reads infra_clusters.json.
func LoadInfraClusters(path string) (InfraClusterFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return InfraClusterFile{}, err
	}
	var file InfraClusterFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return InfraClusterFile{}, err
	}
	for i := range file.Clusters {
		if file.Clusters[i].Score == 0 {
			file.Clusters[i].Score = ScoreInfraCluster(file.Clusters[i])
		}
	}
	return file, nil
}
