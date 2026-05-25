package beads

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestNewExecRunnerDefaultsBinary(t *testing.T) {
	t.Parallel()

	r := newExecRunner("")
	if r.binary != defaultBinary {
		t.Errorf("binary = %q, want %q", r.binary, defaultBinary)
	}
}

func TestExecRunnerCapturesOutput(t *testing.T) {
	t.Parallel()
	requireShell(t)

	r := newExecRunner("/bin/sh")
	result, err := r.run(context.Background(), runnerInvocation{
		args: []string{"-c", "echo out; echo err >&2"},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(string(result.stdout), "out") {
		t.Errorf("stdout = %q, want substring out", result.stdout)
	}
	if !strings.Contains(string(result.stderr), "err") {
		t.Errorf("stderr = %q, want substring err", result.stderr)
	}
}

func TestExecRunnerReturnsExitErrorAndStderr(t *testing.T) {
	t.Parallel()
	requireShell(t)

	r := newExecRunner("/bin/sh")
	result, err := r.run(context.Background(), runnerInvocation{
		args: []string{"-c", "echo bad >&2; exit 7"},
	})
	if err == nil {
		t.Fatal("run: expected error")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("err = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("exit code = %d, want 7", exitErr.ExitCode())
	}
	if !strings.Contains(string(result.stderr), "bad") {
		t.Errorf("stderr = %q, want bad", result.stderr)
	}
}

func TestExecRunnerReturnsContextDeadline(t *testing.T) {
	t.Parallel()
	requireBinary(t, "/bin/sleep")

	r := newExecRunner("/bin/sleep")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	_, err := r.run(ctx, runnerInvocation{args: []string{"5"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline exceeded", err)
	}
}

func requireShell(t *testing.T) {
	t.Helper()
	requireBinary(t, "/bin/sh")
}

func requireBinary(t *testing.T, binary string) {
	t.Helper()
	if _, err := exec.LookPath(binary); err != nil {
		t.Skipf("%s not available", binary)
	}
}
