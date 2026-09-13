package telethon

import (
	"os"
	"strings"
)

// SessionPath reads session file path from TELEGRAM_SESSION, else telegram yaml; default data/runtime/telethon.session.
func SessionPath(configPath string) string {
	if v := strings.TrimSpace(os.Getenv("TELEGRAM_SESSION")); v != "" {
		return v
	}
	path := "data/runtime/telethon.session"
	if configPath == "" {
		return path
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return path
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "session:"); ok {
			if v := strings.TrimSpace(rest); v != "" {
				return v
			}
		}
	}
	return path
}
