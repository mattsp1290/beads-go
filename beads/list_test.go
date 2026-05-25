package beads

import (
	"context"
	"errors"
	"testing"
)

func TestListBuildsArgvAndDecodesIssues(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{
			"--json", "--db=/state", "--actor=agent",
			"list", "--all", "--status=open,closed", "--label=sdk", "-n", "50",
		},
		stdout: []byte(`[
			{"id":"a","status":"open"},
			{"id":"b","status":"closed"}
		]`),
	})
	c, err := NewClient(WithDataDir("/state"), WithActor("agent"), WithListLimit(50))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	issues, err := c.List(context.Background(), WithAll(), WithState("open"), WithState("closed"), WithLabel("sdk"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(issues) != 2 || issues[0].ID != "a" || issues[1].ID != "b" {
		t.Fatalf("issues = %+v", issues)
	}
	rr.assertDone()
}

func TestListReturnsNonNilEmptySlice(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "list"},
		stdout: []byte(`[]`),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	issues, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if issues == nil {
		t.Fatal("issues = nil, want non-nil empty slice")
	}
	if len(issues) != 0 {
		t.Fatalf("len = %d, want 0", len(issues))
	}
	rr.assertDone()
}

func TestListClassifiesRunnerError(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "list", "--status=open"},
		stderr: []byte("validation failed\n"),
		err:    replayExitError(t, 7),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	_, err = c.List(context.Background(), WithState("open"))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("List error = %v, want ErrValidation", err)
	}
	rr.assertDone()
}

func TestListValidatesOptionsBeforeRunner(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}

	_, err = c.List(context.Background(), WithState(""))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("List error = %v, want ErrValidation", err)
	}
}
