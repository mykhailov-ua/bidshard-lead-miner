package forum

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// defaultForumHosts are always allowed; FORUM_HOST_ALLOWLIST adds more.
var defaultForumHosts = []string{
	"affiliatefix.com",
	"stmforum.com",
	"blackhatworld.com",
	"warriorforum.com",
	"afflift.com",
	"gpwa.org",
	"affroom.com",
	"affiliateguarddog.com",
	"digitalpoint.com",
	"cpaelites.com",
	"beermoneyforum.com",
	"wjunction.com",
	"forobeta.com",
	"iamaffiliate.com",
}

var (
	forumHostMu sync.RWMutex
	forumHosts  map[string]struct{}
	hostsReady  bool
)

// ConfigureHosts merges built-in defaults with extra allowlist entries (env or file).
// Safe to call multiple times; last call wins.
func ConfigureHosts(extra []string) {
	seen := make(map[string]struct{}, len(defaultForumHosts)+len(extra))
	for _, h := range defaultForumHosts {
		if norm := normalizeForumHost(h); norm != "" {
			seen[norm] = struct{}{}
		}
	}
	for _, h := range extra {
		if norm := normalizeForumHost(h); norm != "" {
			seen[norm] = struct{}{}
		}
	}
	forumHostMu.Lock()
	forumHosts = seen
	hostsReady = true
	forumHostMu.Unlock()
}

// LoadHostAllowlistFile reads one host per line; # comments and blank lines skipped.
func LoadHostAllowlistFile(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open forum host allowlist: %w", err)
	}
	defer func() { _ = f.Close() }()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read forum host allowlist: %w", err)
	}
	return out, nil
}

// ForumHostCount returns configured allowlist size (defaults + extras).
func ForumHostCount() int {
	ensureHostsConfigured()
	forumHostMu.RLock()
	defer forumHostMu.RUnlock()
	return len(forumHosts)
}

// IsKnownForumHost reports whether host is on the forum crawl allowlist.
// Subdomains match when the allowlist entry is a registrable suffix (forums.gpwa.org -> gpwa.org).
func IsKnownForumHost(host string) bool {
	ensureHostsConfigured()
	host = normalizeForumHost(host)
	if host == "" {
		return false
	}
	forumHostMu.RLock()
	defer forumHostMu.RUnlock()
	if _, ok := forumHosts[host]; ok {
		return true
	}
	for allowed := range forumHosts {
		if strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

func ensureHostsConfigured() {
	forumHostMu.RLock()
	ready := hostsReady
	forumHostMu.RUnlock()
	if ready {
		return
	}
	ConfigureHosts(nil)
}

func normalizeForumHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	return strings.TrimSuffix(host, ".")
}
