package beads

import (
	"context"
	"errors"
	"testing"
)

func TestCommentAppendsNotes(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args: []string{"--json", "--db=/state", "update", "--append-notes=done", "--", "issue-1"},
	})
	c, err := NewClient(WithDataDir("/state"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	if err := c.Comment(context.Background(), " issue-1 ", " done "); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	rr.assertDone()
}

func TestCommentValidatesInputsBeforeRunner(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}

	for _, tc := range []struct {
		name string
		id   string
		body string
	}{
		{"empty id", "", "body"},
		{"dash id", "-bad", "body"},
		{"empty body", "issue-1", " \t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := c.Comment(context.Background(), tc.id, tc.body)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Comment error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestCommentClassifiesRunnerError(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "update", "--append-notes=body", "--", "missing"},
		stderr: []byte("issue missing not found\n"),
		err:    replayExitError(t, 7),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	err = c.Comment(context.Background(), "missing", "body")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Comment error = %v, want ErrNotFound", err)
	}
	rr.assertDone()
}
