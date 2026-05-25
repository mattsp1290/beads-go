package beads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"testing"
)

type replayRun struct {
	args []string
	env  []string
	dir  string

	stdout []byte
	stderr []byte
	err    error
}

type replayRunner struct {
	t           testing.TB
	runs        []replayRun
	index       int
	panicOnCall bool
}

func newReplayRunner(t testing.TB, runs ...replayRun) *replayRunner {
	t.Helper()
	return &replayRunner{t: t, runs: runs}
}

func newPanicRunner(t testing.TB) *replayRunner {
	t.Helper()
	return &replayRunner{t: t, panicOnCall: true}
}

func (r *replayRunner) run(_ context.Context, invocation runnerInvocation) (runnerResult, error) {
	r.t.Helper()
	if r.panicOnCall {
		panic("replayRunner called unexpectedly")
	}
	if r.index >= len(r.runs) {
		r.t.Fatalf("unexpected runner invocation #%d: args=%v env=%v dir=%q", r.index, invocation.args, invocation.env, invocation.dir)
	}
	expected := r.runs[r.index]
	r.index++
	if !slices.Equal(invocation.args, expected.args) {
		r.t.Fatalf("runner invocation #%d args = %v, want %v", r.index-1, invocation.args, expected.args)
	}
	if !slices.Equal(invocation.env, expected.env) {
		r.t.Fatalf("runner invocation #%d env = %v, want %v", r.index-1, invocation.env, expected.env)
	}
	if invocation.dir != expected.dir {
		r.t.Fatalf("runner invocation #%d dir = %q, want %q", r.index-1, invocation.dir, expected.dir)
	}
	return runnerResult{stdout: expected.stdout, stderr: expected.stderr}, expected.err
}

func (r *replayRunner) assertDone() {
	r.t.Helper()
	if r.index != len(r.runs) {
		r.t.Fatalf("runner consumed %d invocation(s), want %d", r.index, len(r.runs))
	}
}

func replayExitError(t testing.TB, code int) error {
	t.Helper()
	if code == 0 {
		return nil
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("exit %d", code))
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	return err
}

func TestReplayRunnerMatchesExpectedInvocation(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "show", "--", "x"},
		env:    []string{"BD_JSON_ENVELOPE=1"},
		dir:    "/repo",
		stdout: []byte("out"),
		stderr: []byte("err"),
	})
	result, err := rr.run(context.Background(), runnerInvocation{
		args:  []string{"--json", "show", "--", "x"},
		env:   []string{"BD_JSON_ENVELOPE=1"},
		dir:   "/repo",
		stdin: io.Reader(nil),
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if string(result.stdout) != "out" || string(result.stderr) != "err" {
		t.Errorf("result = stdout %q stderr %q", result.stdout, result.stderr)
	}
	rr.assertDone()
}

func TestReplayRunnerPanicOnCall(t *testing.T) {
	t.Parallel()

	rr := newPanicRunner(t)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_, _ = rr.run(context.Background(), runnerInvocation{})
}

func TestReplayExitError(t *testing.T) {
	t.Parallel()

	err := replayExitError(t, 7)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("err = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("exit code = %d, want 7", exitErr.ExitCode())
	}
}
