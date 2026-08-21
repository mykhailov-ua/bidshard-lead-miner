package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/spf13/cobra"
)

func TestEffectiveLogFormat(t *testing.T) {
	t.Parallel()
	o := cliOpts{}
	if got := o.effectiveLogFormat(config.Config{Output: "ndjson", LogFormat: "auto"}); got != "text" {
		t.Fatalf("ndjson auto: got %q want text", got)
	}
	o = cliOpts{logFormat: "json"}
	if got := o.effectiveLogFormat(config.Config{Output: "ndjson", LogFormat: "json"}); got != "json" {
		t.Fatalf("explicit json: got %q want json", got)
	}
}

func TestEffectiveLogLevel(t *testing.T) {
	t.Parallel()
	o := cliOpts{quiet: true}
	if got := o.effectiveLogLevel(config.Config{LogLevel: "info"}); got != "warn" {
		t.Fatalf("quiet: got %q want warn", got)
	}
	o = cliOpts{}
	if got := o.effectiveLogLevel(config.Config{Output: "ndjson", LogLevel: "info"}); got != "warn" {
		t.Fatalf("ndjson: got %q want warn", got)
	}
	if got := o.effectiveLogLevel(config.Config{Output: "ndjson", LogLevel: "debug"}); got != "debug" {
		t.Fatalf("explicit debug: got %q want debug", got)
	}
}

func TestJSONFlagSetsOutput(t *testing.T) {
	t.Parallel()
	o := cliOpts{jsonStdout: true}
	cfg := config.Config{Output: "auto"}
	if err := o.apply(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "ndjson" {
		t.Fatalf("output=%q want ndjson", cfg.Output)
	}
}

func TestRootShowsHelpWithoutLegacyFlags(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root help: %v", err)
	}
	if !strings.Contains(buf.String(), "parser scan") {
		t.Fatalf("expected help text, got: %q", buf.String())
	}
}

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
	chdirRepoRoot(t)

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
	chdirRepoRoot(t)

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
