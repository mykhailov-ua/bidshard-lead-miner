package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBlacklist(t *testing.T) {
	dir := t.TempDir()
	domPath := filepath.Join(dir, "domains.txt")
	emailPath := filepath.Join(dir, "emails.txt")

	domContent := "voluum.com\nkeitaro.io\n# comment\n"
	emailContent := "support@voluum.com\ntracker@voluum.com\n"

	if err := os.WriteFile(domPath, []byte(domContent), 0644); err != nil {
		t.Fatalf("write doms: %v", err)
	}
	if err := os.WriteFile(emailPath, []byte(emailContent), 0644); err != nil {
		t.Fatalf("write emails: %v", err)
	}

	if err := LoadBlacklistDomains(domPath); err != nil {
		t.Fatalf("LoadBlacklistDomains failed: %v", err)
	}
	if err := LoadBlacklistEmails(emailPath); err != nil {
		t.Fatalf("LoadBlacklistEmails failed: %v", err)
	}

	if !IsBlacklisted("tracker@voluum.com", "") {
		t.Errorf("expected tracker@voluum.com to be blacklisted")
	}
	if !IsBlacklisted("user@voluum.com", "voluum.com") {
		t.Errorf("expected user@voluum.com / voluum.com domain to be blacklisted")
	}
	if IsBlacklisted("legit@example.com", "example.com") {
		t.Errorf("expected legit@example.com / example.com NOT to be blacklisted")
	}
}

func TestBlacklistRUMailDomains(t *testing.T) {
	dir := t.TempDir()
	domPath := filepath.Join(dir, "domains.txt")
	content := "mail.ru\nyandex.ru\n"
	if err := os.WriteFile(domPath, []byte(content), 0644); err != nil {
		t.Fatalf("write doms: %v", err)
	}
	if err := LoadBlacklistDomains(domPath); err != nil {
		t.Fatalf("LoadBlacklistDomains failed: %v", err)
	}
	for _, email := range []string{"ops@mail.ru", "buyer@yandex.ru"} {
		if !IsBlacklisted(email, "") {
			t.Fatalf("expected %q to be blacklisted", email)
		}
	}
}
