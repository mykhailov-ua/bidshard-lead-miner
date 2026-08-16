package telethon

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Options struct {
	ConfigPath string
	PythonBin  string
	WorkDir    string
	DryRun     bool
	Once       bool
	ExtraEnv   []string
}

func Run(ctx context.Context, opts Options, stdout io.Writer) error {
	if opts.ConfigPath == "" {
		opts.ConfigPath = "config/sources.telegram.yaml"
	}
	if opts.PythonBin == "" {
		opts.PythonBin = "python3"
	}
	if opts.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		opts.WorkDir = wd
	}
	workDir, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return err
	}
	workDir, err = resolveRepoRoot(workDir)
	if err != nil {
		return err
	}
	opts.WorkDir = workDir

	args := []string{"-m", "sources.telegram.scraper", "--config", opts.ConfigPath, "--stdout"}
	if opts.Once {
		args = append(args, "--once")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.CommandContext(ctx, opts.PythonBin, args...)
	cmd.Dir = opts.WorkDir
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	cmd.Env = buildEnv(opts.WorkDir, opts.ExtraEnv)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start telethon sidecar: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		killProcessGroup(cmd)
		select {
		case err := <-done:
			if err != nil {
				return ctx.Err()
			}
		case <-time.After(5 * time.Second):
			killProcessGroupHard(cmd)
		}
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("telethon sidecar: %w", err)
		}
		return nil
	}
}

func buildEnv(workDir string, extra []string) []string {
	env := make([]string, 0, len(os.Environ())+len(extra)+1)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PYTHONPATH=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, extra...)
	env = append(env, "PYTHONPATH="+workDir)
	return env
}

func resolveRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "sources", "telegram", "scraper.py")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start, nil
		}
		dir = parent
	}
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

func killProcessGroupHard(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func DefaultWorkDir() (string, error) {
	return os.Getwd()
}

func ResolveConfig(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return path
}
