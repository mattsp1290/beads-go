package beads

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestSDKMethodsReplayFailureKindCoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  replayRun
		call func(*Client) error
		kind Kind
	}{
		{
			name: "ready timeout",
			run: replayRun{
				args: []string{"--json", "ready"},
				err:  context.DeadlineExceeded,
			},
			call: func(c *Client) error {
				_, err := c.Ready(context.Background())
				return err
			},
			kind: KindTimeout,
		},
		{
			name: "list transport",
			run: replayRun{
				args: []string{"--json", "list"},
				err:  &exec.Error{Name: "bd", Err: exec.ErrNotFound},
			},
			call: func(c *Client) error {
				_, err := c.List(context.Background())
				return err
			},
			kind: KindTransport,
		},
		{
			name: "show not found stderr",
			run: replayRun{
				args:   []string{"--json", "show", "--", "missing"},
				stderr: []byte("issue missing not found\n"),
				err:    replayExitError(t, 7),
			},
			call: func(c *Client) error {
				_, err := c.Show(context.Background(), "missing")
				return err
			},
			kind: KindNotFound,
		},
		{
			name: "close auth stderr",
			run: replayRun{
				args:   []string{"--json", "close", "--reason=done", "--", "x"},
				stderr: []byte("forbidden\n"),
				err:    replayExitError(t, 7),
			},
			call: func(c *Client) error {
				return c.Close(context.Background(), "x", "done")
			},
			kind: KindAuthFailed,
		},
		{
			name: "comment exit status",
			run: replayRun{
				args:   []string{"--json", "update", "--append-notes=body", "--", "x"},
				stderr: []byte("opaque failure\n"),
				err:    replayExitError(t, 7),
			},
			call: func(c *Client) error {
				return c.Comment(context.Background(), "x", "body")
			},
			kind: KindExit,
		},
		{
			name: "transition validation stderr",
			run: replayRun{
				args:   []string{"--json", "update", "--status=bad", "--", "x"},
				stderr: []byte("unknown status bad\n"),
				err:    replayExitError(t, 7),
			},
			call: func(c *Client) error {
				return c.Transition(context.Background(), "x", "bad")
			},
			kind: KindValidation,
		},
		{
			name: "ready bad json",
			run: replayRun{
				args:   []string{"--json", "ready"},
				stdout: []byte(`{"schema_version":99,"data":[]}`),
			},
			call: func(c *Client) error {
				_, err := c.Ready(context.Background())
				return err
			},
			kind: KindBadResponse,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := newReplayRunner(t, tc.run)
			c, err := NewClient()
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			c.transport = &execTransport{config: c.config, runner: rr}

			err = tc.call(c)
			if !errors.Is(err, &Error{Kind: tc.kind}) {
				t.Fatalf("error = %v, want kind %s", err, tc.kind)
			}
			if tc.kind == KindExit {
				var be *Error
				if !errors.As(err, &be) {
					t.Fatalf("errors.As = false")
				}
				if be.Status != 7 {
					t.Fatalf("exit status = %d, want 7", be.Status)
				}
			}
			rr.assertDone()
		})
	}
}

func TestSDKMethodsDecodeJSONEnvelopePayloads(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  replayRun
		call func(*Client) (Issue, error)
	}{
		{
			name: "ready",
			run: replayRun{
				args:   []string{"--json", "ready", "-n", "0"},
				stdout: []byte(`{"schema_version":1,"data":[{"id":"issue-1","status":"open","title":"Ready"}]}`),
			},
			call: func(c *Client) (Issue, error) {
				issues, err := c.Ready(context.Background(), WithAll())
				if err != nil || len(issues) == 0 {
					return Issue{}, err
				}
				return issues[0], nil
			},
		},
		{
			name: "list",
			run: replayRun{
				args:   []string{"--json", "list", "--all"},
				stdout: []byte(`{"schema_version":1,"data":[{"id":"issue-2","status":"closed","title":"Listed"}]}`),
			},
			call: func(c *Client) (Issue, error) {
				issues, err := c.List(context.Background(), WithAll())
				if err != nil || len(issues) == 0 {
					return Issue{}, err
				}
				return issues[0], nil
			},
		},
		{
			name: "show",
			run: replayRun{
				args:   []string{"--json", "show", "--", "issue-3"},
				stdout: []byte(`{"schema_version":1,"data":{"id":"issue-3","status":"open","title":"Shown"}}`),
			},
			call: func(c *Client) (Issue, error) {
				return c.Show(context.Background(), "issue-3")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := newReplayRunner(t, tc.run)
			c, err := NewClient()
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			c.transport = &execTransport{config: c.config, runner: rr}

			issue, err := tc.call(c)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if issue.ID == "" || issue.Status == "" {
				t.Fatalf("issue = %+v, want decoded id and status", issue)
			}
			if len(issue.RawJSON) == 0 || issue.RawJSON[0] != '{' {
				t.Fatalf("RawJSON = %s, want preserved issue object", issue.RawJSON)
			}
			rr.assertDone()
		})
	}
}

func TestUnsupportedKindSentinel(t *testing.T) {
	t.Parallel()

	err := &Error{Op: "LinkPR", Kind: KindUnsupported, Err: ErrUnsupported}
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("errors.Is(%v, ErrUnsupported) = false", err)
	}
}
