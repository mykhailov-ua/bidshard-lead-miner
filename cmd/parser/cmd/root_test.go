package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandTree(t *testing.T) {
	want := []string{
		"run", "scan", "telegram", "ingest", "version", "sources", "config",
	}
	got := rootCmd.Commands()
	if len(got) < len(want) {
		t.Fatalf("got %d commands, want at least %d", len(got), len(want))
	}

	byName := make(map[string]*cobra.Command, len(got))
	for _, c := range got {
		byName[c.Name()] = c
	}
	for _, name := range want {
		c, ok := byName[name]
		if !ok {
			t.Fatalf("missing command %q", name)
		}
		if c.Short == "" {
			t.Fatalf("command %q has empty Short", name)
		}
	}
}

func TestHelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "scan") {
		t.Fatalf("help output missing scan command:\n%s", out)
	}
	if !strings.Contains(out, "config") {
		t.Fatalf("help output missing config command:\n%s", out)
	}
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(buf.String(), "parser") {
		t.Fatalf("unexpected version output: %q", buf.String())
	}
}

func TestConfigCheckStub(t *testing.T) {
	t.Chdir("../../../")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config check: %v\noutput: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "ok") {
		t.Fatalf("expected ok in output: %q", buf.String())
	}
}

func TestLegacyScanOnceFlag(t *testing.T) {
	t.Chdir("../../../")

	f := rootCmd.Flags().Lookup("scan-once")
	if f == nil {
		t.Fatal("missing legacy --scan-once flag")
	}
}

func TestNormalizeLegacyArgs(t *testing.T) {
	cases := map[string]string{
		"-scan-once":    "--scan-once",
		"-output=quiet": "--output=quiet",
		"-source=stub":  "--source=stub",
		"-h":            "-h",
		"scan":          "scan",
		"--source=stub": "--source=stub",
	}
	for in, want := range cases {
		if got := normalizeLegacyArg(in); got != want {
			t.Fatalf("normalizeLegacyArg(%q) = %q, want %q", in, got, want)
		}
	}
}
