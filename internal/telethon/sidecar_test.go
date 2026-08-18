package telethon

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestKillProcessGroupTerminatesChild(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	cmd := exec.Command("python3", "-c", "import time; time.sleep(60)")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	killProcessGroup(cmd)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected terminated process")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("child process did not exit after SIGTERM to group")
	}
}

func TestDryRunEmitsTelegramNDJSON(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		err := Run(ctx, Options{DryRun: true, Once: true, WorkDir: mustWorkDir(t)}, pw)
		_ = pw.Close()
		errCh <- err
	}()

	out, readErr := io.ReadAll(pr)
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("sidecar: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout")
	}

	if len(out) == 0 || !strings.Contains(string(out), "telegram:@") {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func mustWorkDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}
