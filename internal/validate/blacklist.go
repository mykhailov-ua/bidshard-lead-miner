package validate

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	blacklistMu        sync.RWMutex
	blacklistedDomains = make(map[string]struct{})
	blacklistedEmails  = make(map[string]struct{})
)

// LoadBlacklistDomains loads blacklisted domains from a line-based file.
func LoadBlacklistDomains(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open blacklist domains: %w", err)
	}
	defer func() { _ = f.Close() }()

	blacklistMu.Lock()
	defer blacklistMu.Unlock()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		domain := normalizeBlacklistEntry(sc.Text())
		if domain == "" {
			continue
		}
		blacklistedDomains[domain] = struct{}{}
	}
	return sc.Err()
}

// LoadBlacklistEmails loads blacklisted emails from a line-based file.
func LoadBlacklistEmails(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open blacklist emails: %w", err)
	}
	defer func() { _ = f.Close() }()

	blacklistMu.Lock()
	defer blacklistMu.Unlock()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		email := normalizeBlacklistEntry(sc.Text())
		if email == "" {
			continue
		}
		blacklistedEmails[email] = struct{}{}
	}
	return sc.Err()
}

func normalizeBlacklistEntry(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	return strings.ToLower(line)
}

func BlacklistDomainCount() int {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	return len(blacklistedDomains)
}

func BlacklistEmailCount() int {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	return len(blacklistedEmails)
}

// IsBlacklisted reports whether email or domain is blacklisted.
func IsBlacklisted(email, domain string) bool {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()

	emailClean := strings.ToLower(strings.TrimSpace(email))
	if emailClean != "" {
		if _, ok := blacklistedEmails[emailClean]; ok {
			return true
		}
		// check if email domain is blacklisted
		if idx := strings.Index(emailClean, "@"); idx >= 0 {
			emailDomain := emailClean[idx+1:]
			if _, ok := blacklistedDomains[emailDomain]; ok {
				return true
			}
		}
	}

	domainClean := strings.ToLower(strings.TrimSpace(domain))
	if domainClean != "" {
		if _, ok := blacklistedDomains[domainClean]; ok {
			return true
		}
	}

	return false
}

// IsBlacklistedDomain reports whether a hostname is on the crawl blacklist.
func IsBlacklistedDomain(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "www.")
	if domain == "" {
		return false
	}
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	if _, ok := blacklistedDomains[domain]; ok {
		return true
	}
	for blocked := range blacklistedDomains {
		if strings.HasSuffix(domain, "."+blocked) {
			return true
		}
	}
	return false
}
