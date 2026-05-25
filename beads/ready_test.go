package beads

import (
	"context"
	"errors"
	"testing"
)

func TestReadyBuildsArgvAndDecodesIssues(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{
			"--json", "--db=/state", "--actor=agent",
			"ready", "--label=sdk", "--label=p0", "-n", "25",
		},
		stdout: []byte(`[
			{"id":"a","status":"open"},
			{"id":"b","status":"closed"}
		]`),
	})
	c, err := NewClient(WithDataDir("/state"), WithActor("agent"), WithListLimit(25))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	issues, err := c.Ready(context.Background(), WithLabel("sdk"), WithLabel("p0"))
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("len = %d, want 2", len(issues))
	}
	if issues[0].ID != "a" || issues[1].ID != "b" {
		t.Errorf("issues = %+v", issues)
	}
	rr.assertDone()
}

func TestReadyWithAllOverridesListLimit(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "ready", "-n", "0"},
		stdout: []byte(`[]`),
	})
	c, err := NewClient(WithListLimit(25))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	issues, err := c.Ready(context.Background(), WithAll())
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if issues == nil {
		t.Fatal("issues = nil, want non-nil empty slice")
	}
	if len(issues) != 0 {
		t.Fatalf("len = %d, want 0", len(issues))
	}
	rr.assertDone()
}

func TestReadyFiltersStatesClientSide(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{"--json", "ready"},
		stdout: []byte(`[
			{"id":"a","status":"open"},
			{"id":"b","status":"in_progress"}
		]`),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	issues, err := c.Ready(context.Background(), WithState("in_progress"))
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != "b" {
		t.Fatalf("issues = %+v, want only b", issues)
	}
	rr.assertDone()
}

func TestReadyClassifiesRunnerError(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "ready"},
		stderr: []byte("permission denied\n"),
		err:    replayExitError(t, 7),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	_, err = c.Ready(context.Background())
	if !errors.Is(err, &Error{Kind: KindAuthFailed}) {
		t.Fatalf("Ready error = %v, want KindAuthFailed", err)
	}
	rr.assertDone()
}

func TestReadyValidatesOptionsBeforeRunner(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}

	_, err = c.Ready(context.Background(), WithLabel(" "))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Ready error = %v, want ErrValidation", err)
	}
}
