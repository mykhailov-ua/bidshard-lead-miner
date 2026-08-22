package discover

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

type DorkRankRow struct {
	Query       string  `json:"query"`
	Accepted    int     `json:"accepted"`
	Junk        int     `json:"junk"`
	AcceptRate  float64 `json:"accept_rate"`
	ChannelHits int     `json:"channel_hits"`
}

type channelFile struct {
	Channels []struct {
		Username string `json:"username"`
		Query    string `json:"query"`
	} `json:"channels"`
}

type dorkIndex struct {
	usernameToQuery map[string]string
	hitsPerQuery    map[string]int
}

// WriteDorkRankReport ranks SERP dorks by accept/junk ratio from source_stats.
func WriteDorkRankReport(channelsPath, outDir string, stats []sink.SourceStatsDoc) (string, error) {
	if outDir == "" {
		outDir = "data/suggestions"
	}
	idx, err := loadDorkIndex(channelsPath)
	if err != nil {
		slog.Warn("dork rank: channels file unreadable, telegram stats skipped", "path", channelsPath, "error", err)
		idx = dorkIndex{
			usernameToQuery: map[string]string{},
			hitsPerQuery:    map[string]int{},
		}
	}

	byDork := map[string]*DorkRankRow{}
	for _, row := range stats {
		source := strings.TrimSpace(row.Source)
		if source == "" {
			continue
		}
		q := matchTelegramDork(source, idx.usernameToQuery)
		if q == "" {
			continue
		}
		entry := byDork[q]
		if entry == nil {
			entry = &DorkRankRow{Query: q}
			byDork[q] = entry
		}
		entry.Accepted += row.Accepted
		entry.Junk += row.Junk
	}
	rows := make([]DorkRankRow, 0, len(byDork))
	for q, row := range byDork {
		total := row.Accepted + row.Junk
		if total > 0 {
			row.AcceptRate = float64(row.Accepted) / float64(total)
		}
		row.ChannelHits = idx.hitsPerQuery[q]
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].AcceptRate == rows[j].AcceptRate {
			return rows[i].Accepted > rows[j].Accepted
		}
		return rows[i].AcceptRate > rows[j].AcceptRate
	})

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102")
	path := filepath.Join(outDir, "dork_rank_"+stamp+".json")
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func loadDorkIndex(channelsPath string) (dorkIndex, error) {
	idx := dorkIndex{
		usernameToQuery: map[string]string{},
		hitsPerQuery:    map[string]int{},
	}
	if channelsPath == "" {
		return idx, fmt.Errorf("channels path empty")
	}
	raw, err := os.ReadFile(channelsPath)
	if err != nil {
		return idx, err
	}
	var f channelFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return idx, err
	}
	for _, ch := range f.Channels {
		q := strings.TrimSpace(ch.Query)
		u := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ch.Username), "@"))
		if q != "" {
			idx.hitsPerQuery[q]++
		}
		if u != "" && q != "" {
			idx.usernameToQuery[u] = q
		}
	}
	return idx, nil
}

// matchTelegramDork maps telegram:username sources to the SERP query that discovered them.
func matchTelegramDork(source string, usernameToQuery map[string]string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if !strings.HasPrefix(source, "telegram:") {
		return ""
	}
	slug := strings.TrimPrefix(source, "telegram:")
	if i := strings.Index(slug, "/"); i >= 0 {
		slug = slug[:i]
	}
	slug = strings.TrimPrefix(slug, "@")
	return usernameToQuery[slug]
}
