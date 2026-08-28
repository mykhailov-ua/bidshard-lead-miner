package pretty

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestColorEnabledRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "")
	if ColorEnabled(os.Stdout) {
		t.Fatal("expected colors disabled when NO_COLOR is set")
	}
}

func TestStatusOKPlain(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	StatusOK(&buf, true, "loaded %s", "forum")
	got := buf.String()
	if !strings.Contains(got, "ok  loaded forum") {
		t.Fatalf("unexpected output: %q", got)
	}
	if strings.Contains(got, "\033[") {
		t.Fatalf("expected no ANSI escapes: %q", got)
	}
}

func TestPrintTableHeader(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	PrintTable(&buf, true, []string{"name", "value"}, [][]string{{"foo", "bar"}})
	got := buf.String()
	if !strings.Contains(got, "name") || !strings.Contains(got, "foo") {
		t.Fatalf("unexpected table: %q", got)
	}
}
