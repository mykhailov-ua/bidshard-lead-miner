package config

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv loads unset variables from the first .env found walking up from cwd.
// Process environment always wins; this does not search from executable path.
func LoadDotEnv() {
	loadDotEnv()
}

// loadDotEnv sets unset variables from .env in cwd or any parent directory.
func loadDotEnv() {
	start, err := os.Getwd()
	if err != nil {
		return
	}
	dir := start
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			applyEnvFile(path)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func applyEnvFile(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Do not override variables already set in the process environment (including empty).
		if key == "" {
			continue
		}
		if _, ok := os.LookupEnv(key); ok {
			continue
		}
		val = strings.Trim(val, `"'`)
		_ = os.Setenv(key, val)
	}
}
