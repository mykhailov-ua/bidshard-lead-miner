package validate

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	disposableMu sync.RWMutex
	// disposableDomains is merged from built-ins and DISPOSABLE_DOMAINS_PATH at startup.
	// RWMutex allows safe reload if hot-reload is added later; AcceptEmail reads via IsDisposableDomain.
	disposableDomains = map[string]struct{}{
		"mailinator.com":    {},
		"guerrillamail.com": {},
		"tempmail.com":      {},
		"10minutemail.com":  {},
		"throwaway.email":   {},
		"example.com":       {},
		"test.com":          {},
		"yopmail.com":       {},
		"sharklasers.com":   {},
		"trashmail.com":     {},
		"getnada.com":       {},
		"temp-mail.org":     {},
		"dispostable.com":   {},
		"maildrop.cc":       {},
		"fakeinbox.com":     {},
		"spamgourmet.com":   {},
		"mailnesia.com":     {},
		"tempail.com":       {},
		"moakt.com":         {},
		"emailondeck.com":   {},
		"mohmal.com":        {},
		"burnermail.io":     {},
		"mailcatch.com":     {},
		"mytemp.email":      {},
		"tmpmail.net":       {},
		"tmpmail.org":       {},
		"mail-temp.com":     {},
	}
)

// LoadDisposableDomains merges domains from a line-based file (one domain per line, # comments).
// Missing file is not an error. Safe to call at startup or reload under disposableMu.
func LoadDisposableDomains(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open disposable domains: %w", err)
	}
	defer func() { _ = f.Close() }()

	disposableMu.Lock()
	defer disposableMu.Unlock()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		domain := normalizeDisposableDomain(sc.Text())
		if domain == "" {
			continue
		}
		disposableDomains[domain] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read disposable domains: %w", err)
	}
	return nil
}

func normalizeDisposableDomain(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	line = strings.ToLower(line)
	return strings.TrimPrefix(line, "@")
}

func DisposableDomainCount() int {
	disposableMu.RLock()
	defer disposableMu.RUnlock()
	return len(disposableDomains)
}

func IsDisposableDomain(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	disposableMu.RLock()
	defer disposableMu.RUnlock()
	_, ok := disposableDomains[domain]
	return ok
}
