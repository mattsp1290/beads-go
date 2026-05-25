package beads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"
)

type runner interface {
	run(ctx context.Context, invocation runnerInvocation) (runnerResult, error)
}

type runnerInvocation struct {
	args  []string
	env   []string
	dir   string
	stdin io.Reader
}

type runnerResult struct {
	stdout []byte
	stderr []byte
}

type execRunner struct {
	binary string
}

func newExecRunner(binary string) *execRunner {
	if binary == "" {
		binary = defaultBinary
	}
	return &execRunner{binary: binary}
}

func (r *execRunner) run(ctx context.Context, invocation runnerInvocation) (runnerResult, error) {
	const maxAttempts = 3
	var last runnerResult
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		last, err = r.runOnce(ctx, invocation)
		if !errors.Is(err, syscall.ETXTBSY) {
			return last, err
		}
		if ctx.Err() != nil {
			return last, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	return last, err
}

func (r *execRunner) runOnce(ctx context.Context, invocation runnerInvocation) (runnerResult, error) {
	// G204: exec with non-constant binary and argv is intentional. The binary
	// comes from operator configuration, argv is built internally by Client
	// methods, no shell is invoked, and every public method that accepts a
	// positional issue ID must use requireID plus a literal "--" separator via
	// appendPositionalID before the ID reaches this runner.
	//#nosec G204
	cmd := exec.CommandContext(ctx, r.binary, invocation.args...)
	if invocation.stdin != nil {
		cmd.Stdin = invocation.stdin
	}
	if invocation.dir != "" {
		cmd.Dir = invocation.dir
	}
	if invocation.env != nil {
		cmd.Env = invocation.env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil && err != nil {
		if errors.Is(ctxErr, context.Canceled) || errors.Is(ctxErr, context.DeadlineExceeded) {
			return runnerResult{stdout: stdout.Bytes(), stderr: stderr.Bytes()}, ctxErr
		}
	}
	return runnerResult{stdout: stdout.Bytes(), stderr: stderr.Bytes()}, err
}
