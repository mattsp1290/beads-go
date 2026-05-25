package beads

import (
	"context"
	"errors"
	"testing"
)

func TestShowFetchesIssue(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "--db=/state", "show", "--", "issue-1"},
		stdout: []byte(`[{"id":"issue-1","status":"open","title":"T"}]`),
	})
	c, err := NewClient(WithDataDir("/state"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	issue, err := c.Show(context.Background(), " issue-1 ")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if issue.ID != "issue-1" || issue.Title != "T" {
		t.Errorf("issue = %+v", issue)
	}
	rr.assertDone()
}

func TestShowEmptyResultIsNotFound(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "show", "--", "missing"},
		stdout: []byte(`[]`),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	_, err = c.Show(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Show error = %v, want ErrNotFound", err)
	}
	rr.assertDone()
}

func TestShowClassifiesNotFoundStderr(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "show", "--", "missing"},
		stderr: []byte("issue missing not found\n"),
		err:    replayExitError(t, 7),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	_, err = c.Show(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Show error = %v, want ErrNotFound", err)
	}
	rr.assertDone()
}

func TestShowRejectsMismatchedID(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "show", "--", "issue-1"},
		stdout: []byte(`[{"id":"other","status":"open"}]`),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	_, err = c.Show(context.Background(), "issue-1")
	if !errors.Is(err, &Error{Kind: KindBadResponse}) {
		t.Fatalf("Show error = %v, want KindBadResponse", err)
	}
	rr.assertDone()
}

func TestShowValidatesIDBeforeRunner(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}

	for _, id := range []string{"", " \t", "-bad"} {
		_, err := c.Show(context.Background(), id)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("Show(%q) error = %v, want ErrValidation", id, err)
		}
	}
}
