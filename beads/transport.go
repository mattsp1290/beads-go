package beads

import (
	"context"
	"io"
)

type transport interface {
	execute(ctx context.Context, call bdCall) (bdOutput, error)
}

type bdCall struct {
	op    string
	args  []string
	env   []string
	dir   string
	stdin io.Reader
}

type bdOutput struct {
	stdout []byte
	stderr []byte
}

type execTransport struct {
	config clientConfig
	runner runner
}

func newExecTransport(cfg clientConfig) *execTransport {
	return &execTransport{
		config: cfg,
		runner: newExecRunner(cfg.binary),
	}
}

func (t *execTransport) execute(ctx context.Context, call bdCall) (bdOutput, error) {
	args := t.baseArgs(call.args)
	result, err := t.runner.run(ctx, runnerInvocation{
		args:  args,
		env:   call.env,
		dir:   call.dir,
		stdin: call.stdin,
	})
	return bdOutput(result), err
}

func (t *execTransport) baseArgs(subcommand []string) []string {
	args := []string{"--json"}
	if t.config.dataDir != "" {
		args = append(args, "--db="+t.config.dataDir)
	}
	if t.config.actor != "" {
		args = append(args, "--actor="+t.config.actor)
	}
	args = append(args, subcommand...)
	return args
}
