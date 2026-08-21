package lander

import "testing"

func TestSPAShellFingerprintSameShell(t *testing.T) {
	shell := "<!DOCTYPE html><html><head><title>App</title></head><body><div id=\"root\"></div></body></html>"
	pad := stringsRepeat("x", 500)
	body1 := shell + pad
	body2 := shell + pad

	fp1 := SPAShellFingerprint(body1)
	fp2 := SPAShellFingerprint(body2)
	if fp1 == "" || fp1 != fp2 {
		t.Fatalf("expected same fingerprint, got %q vs %q", fp1, fp2)
	}
	if !IsSPA404Shell(404, body1) {
		t.Fatal("expected spa 404 shell")
	}
}

func TestIsSPA404ShellRejectsTiny404(t *testing.T) {
	if IsSPA404Shell(404, "<html><body>404</body></html>") {
		t.Fatal("tiny 404 should not be spa shell")
	}
	if IsSPA404Shell(200, stringsRepeat("a", 1000)) {
		t.Fatal("200 should not be spa 404")
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
