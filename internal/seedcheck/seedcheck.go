package seedcheck

import (
	"os"
	"strings"
)

const ProfileProd = "prod"

// Profile returns PARSER_SEED_PROFILE (dev|prod); default dev.
func Profile() string {
	p := strings.ToLower(strings.TrimSpace(os.Getenv("PARSER_SEED_PROFILE")))
	if p == ProfileProd {
		return ProfileProd
	}
	return "dev"
}

// FixtureMarkers in seed files that are invalid for production crawl.
func FixtureMarkers() []string {
	return []string{".example", "forum-fixture.test"}
}

// FileLooksLikeFixture reports dev/CI placeholder content in a seed file.
func FileLooksLikeFixture(path string) (bool, string) {
	if path == "" {
		return false, ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, ""
	}
	content := strings.ToLower(string(data))
	for _, marker := range FixtureMarkers() {
		if strings.Contains(content, marker) {
			return true, marker
		}
	}
	return false, ""
}

// CountDataRows counts non-comment, non-header CSV rows in a seed file.
func CountDataRows(path string) (int, error) {
	if path == "" {
		return 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "url,") ||
			strings.HasPrefix(strings.ToLower(line), "domain,") {
			continue
		}
		count++
	}
	return count, nil
}

// ProdSeedMinRows is the minimum data rows expected in prod seed files.
var ProdSeedMinRows = map[string]int{
	"forum":  5,
	"supply": 5,
	"lander": 2,
}

func ProdSeedPaths() map[string]string {
	return map[string]string{
		"SUPPLY_SEED_PATH": "data/seeds/domains.prod.csv",
		"LANDER_SEED_PATH": "data/seeds/lander_urls.prod.csv",
		"FORUM_SEED_PATH":  "data/seeds/forum_threads.live.csv",
	}
}
