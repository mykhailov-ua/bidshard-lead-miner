package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadProxyURLsFromFile reads proxy endpoints (one per line).
// Formats: user:pass@host:port or http(s)://user:pass@host:port
func LoadProxyURLsFromFile(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("proxy list path empty")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var out []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := normalizeProxyLine(line)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, u)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no proxy entries", path)
	}
	return out, nil
}

func normalizeProxyLine(line string) (string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("empty proxy line")
	}
	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		return line, nil
	}
	if strings.Contains(line, "://") {
		return "", fmt.Errorf("unsupported proxy scheme in %q", redactProxyURL(line))
	}
	return "http://" + line, nil
}
