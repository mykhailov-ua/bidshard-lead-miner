package forum

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fixtureHost = "forum-fixture.test"

var fixtureBySlug = map[string]string{
	"voluum":   "stm_thread.html",
	"postback": "postback_thread.html",
	"keitaro":  "keitaro_thread.html",
	"binom":    "binom_thread.html",
	"redtrack": "redtrack_thread.html",
}

func isFixtureURL(rawURL string) bool {
	return hostFromURL(rawURL) == fixtureHost
}

func loadFixtureHTML(rawURL string) (string, error) {
	slug := strings.ToLower(rawURL)
	name := "stm_thread.html"
	for key, file := range fixtureBySlug {
		if strings.Contains(slug, key) {
			name = file
			break
		}
	}
	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "testdata", "forum", name)
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("forum fixture %s: %w", name, err)
	}
	return string(body), nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "testdata", "forum", "stm_thread.html")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("testdata/forum not found from %s", wd)
		}
		dir = parent
	}
}
