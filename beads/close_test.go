package beads

import (
	"context"
	"errors"
	"testing"
)

func TestCloseWithReason(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{"--json", "--actor=agent", "close", "--reason=done", "--", "issue-1"},
	})
	c, err := NewClient(WithActor("agent"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	if err := c.Close(context.Background(), " issue-1 ", " done "); err != nil {
		t.Fatalf("Close: %v", err)
	}
	rr.assertDone()
}

func TestCloseOmitEmptyReason(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{"--json", "close", "--", "issue-1"},
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	if err := c.Close(context.Background(), "issue-1", " \t"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	rr.assertDone()
}

func TestCloseValidatesIDBeforeRunner(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}

	for _, id := range []string{"", " \t", "-bad"} {
		err := c.Close(context.Background(), id, "done")
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("Close(%q) error = %v, want ErrValidation", id, err)
		}
	}
}

func TestCloseClassifiesRunnerError(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "close", "--reason=done", "--", "issue-1"},
		stderr: []byte("permission denied\n"),
		err:    replayExitError(t, 7),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	err = c.Close(context.Background(), "issue-1", "done")
	if !errors.Is(err, &Error{Kind: KindAuthFailed}) {
		t.Fatalf("Close error = %v, want KindAuthFailed", err)
	}
	rr.assertDone()
}
