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
		defer pw.Close()
		wd, _ := os.Getwd()
		errCh <- Run(ctx, Options{DryRun: true, Once: true, WorkDir: wd}, pw)
	}()

	buf := make([]byte, 4096)
	n, readErr := pr.Read(buf)
	_ = pr.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("sidecar: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout")
	}

	if readErr != nil && readErr != io.EOF {
		t.Fatalf("read: %v", readErr)
	}
	if n == 0 || !strings.Contains(string(buf[:n]), "telegram:@") {
		t.Fatalf("unexpected output: %q", string(buf[:n]))
	}
}
