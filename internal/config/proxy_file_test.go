package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProxyURLsFromFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "proxy.list")
	content := "# residential\nuser:pass@1.2.3.4:10000\nhttp://u:p@5.6.7.8:8080\n\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadProxyURLsFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "http://user:pass@1.2.3.4:10000" {
		t.Fatalf("got %v", got)
	}
}
