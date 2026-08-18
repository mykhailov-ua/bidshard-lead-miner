package validate

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadDisposableDomains merges domains from a line-based file (one domain per line, # comments).
// Safe to call once at startup; missing file is not an error.
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
	defer f.Close()

	added := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		domain := normalizeDisposableDomain(sc.Text())
		if domain == "" {
			continue
		}
		disposableDomains[domain] = struct{}{}
		added++
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
	return len(disposableDomains)
}

func IsDisposableDomain(domain string) bool {
	_, ok := disposableDomains[strings.ToLower(strings.TrimSpace(domain))]
	return ok
}
