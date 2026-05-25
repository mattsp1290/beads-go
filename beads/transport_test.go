package beads

import (
	"context"
	"slices"
	"strings"
	"testing"
)

type recordingRunner struct {
	invocation runnerInvocation

	result runnerResult
	err    error
}

func (r *recordingRunner) run(_ context.Context, invocation runnerInvocation) (runnerResult, error) {
	r.invocation = runnerInvocation{
		args:  append([]string(nil), invocation.args...),
		env:   append([]string(nil), invocation.env...),
		dir:   invocation.dir,
		stdin: invocation.stdin,
	}
	return r.result, r.err
}

func TestNewClientInitializesExecTransport(t *testing.T) {
	t.Parallel()

	c, err := NewClient(WithBinary("/bin/bd"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	et, ok := c.transport.(*execTransport)
	if !ok {
		t.Fatalf("transport = %T, want *execTransport", c.transport)
	}
	if et.config.binary != "/bin/bd" {
		t.Errorf("transport binary = %q", et.config.binary)
	}
	if _, ok := et.runner.(*execRunner); !ok {
		t.Errorf("runner = %T, want *execRunner", et.runner)
	}
}

func TestExecTransportPrependsGlobalArgs(t *testing.T) {
	t.Parallel()

	rr := &recordingRunner{result: runnerResult{stdout: []byte("out"), stderr: []byte("err")}}
	tp := &execTransport{
		config: clientConfig{
			binary:  defaultBinary,
			dataDir: "/state",
			actor:   "agent",
		},
		runner: rr,
	}

	stdin := strings.NewReader("input")
	out, err := tp.execute(context.Background(), bdCall{
		op:    "Close",
		args:  []string{"close", "--", "issue-1"},
		env:   []string{"BD_JSON_ENVELOPE=1"},
		dir:   "/repo",
		stdin: stdin,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	wantArgs := []string{"--json", "--db=/state", "--actor=agent", "close", "--", "issue-1"}
	if !slices.Equal(rr.invocation.args, wantArgs) {
		t.Errorf("args = %v, want %v", rr.invocation.args, wantArgs)
	}
	if !slices.Equal(rr.invocation.env, []string{"BD_JSON_ENVELOPE=1"}) {
		t.Errorf("env = %v", rr.invocation.env)
	}
	if rr.invocation.dir != "/repo" {
		t.Errorf("dir = %q", rr.invocation.dir)
	}
	if rr.invocation.stdin != stdin {
		t.Error("stdin was not forwarded")
	}
	if string(out.stdout) != "out" || string(out.stderr) != "err" {
		t.Errorf("output = stdout %q stderr %q", out.stdout, out.stderr)
	}
}

func TestExecTransportBaseArgsDoesNotMutateSubcommand(t *testing.T) {
	t.Parallel()

	tp := &execTransport{config: clientConfig{binary: defaultBinary}}
	sub := []string{"show", "--", "issue-1"}
	got := tp.baseArgs(sub)
	want := []string{"--json", "show", "--", "issue-1"}
	if !slices.Equal(got, want) {
		t.Errorf("baseArgs = %v, want %v", got, want)
	}
	if !slices.Equal(sub, []string{"show", "--", "issue-1"}) {
		t.Errorf("subcommand mutated: %v", sub)
	}
}
