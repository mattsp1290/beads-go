package beads

import (
	"context"
	"errors"
	"testing"
)

func TestTransitionCallsBDUpdateStatus(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{"--json", "--db=/state", "--actor=agent", "update", "--status=in_progress", "--", "issue-1"},
	})
	c, err := NewClient(WithDataDir("/state"), WithActor("agent"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	if err := c.Transition(context.Background(), " issue-1 ", " in_progress "); err != nil {
		t.Fatalf("Transition: %v", err)
	}
	rr.assertDone()
}

func TestTransitionValidatesInputsBeforeRunner(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}

	for _, tc := range []struct {
		name  string
		id    string
		state string
	}{
		{"empty id", "", "open"},
		{"dash id", "-bad", "open"},
		{"empty state", "issue-1", " \t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := c.Transition(context.Background(), tc.id, tc.state)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Transition error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestTransitionClassifiesValidationStderr(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "update", "--status=bad", "--", "issue-1"},
		stderr: []byte("invalid status: bad\n"),
		err:    replayExitError(t, 7),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	err = c.Transition(context.Background(), "issue-1", "bad")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Transition error = %v, want ErrValidation", err)
	}
	rr.assertDone()
}
