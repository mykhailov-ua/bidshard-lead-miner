package telethon

import (
	"os"
	"strings"
)

// SessionPath reads session file path from telegram yaml; default data/runtime/telethon.session.
func SessionPath(configPath string) string {
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
