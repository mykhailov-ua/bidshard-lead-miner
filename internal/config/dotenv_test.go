package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("PARSER_DOTENV_TEST=from_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig, had := os.LookupEnv("PARSER_DOTENV_TEST")
	defer func() {
		if had {
			_ = os.Setenv("PARSER_DOTENV_TEST", orig)
		} else {
			_ = os.Unsetenv("PARSER_DOTENV_TEST")
		}
	}()
	_ = os.Unsetenv("PARSER_DOTENV_TEST")

	applyEnvFile(path)
	if got := os.Getenv("PARSER_DOTENV_TEST"); got != "from_file" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyEnvFileRespectsEmptyEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("PARSER_DOTENV_EMPTY=from_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PARSER_DOTENV_EMPTY", "")
	applyEnvFile(path)
	if got := os.Getenv("PARSER_DOTENV_EMPTY"); got != "" {
		t.Fatalf("got %q want empty (explicit env wins)", got)
	}
}
