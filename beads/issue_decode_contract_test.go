package beads

import (
	"errors"
	"strings"
	"testing"
)

func TestIssueDecodeContractReadyListAndShowPayloads(t *testing.T) {
	t.Parallel()

	t.Run("ready array unknown fields and ready dependency shape", func(t *testing.T) {
		t.Parallel()
		issues, err := decodeIssues("Ready", []byte(`[
			{
				"id":"ready-1",
				"status":"open",
				"unknown_ready_field":{"nested":true},
				"dependencies":[{"issue_id":"ready-1","depends_on_id":"blocker-1","type":"blocks"}]
			}
		]`))
		if err != nil {
			t.Fatalf("decodeIssues: %v", err)
		}
		if got := issues[0].Dependencies[0].DependsOnID; got != "blocker-1" {
			t.Errorf("DependsOnID = %q", got)
		}
		if !strings.Contains(string(issues[0].RawJSON), "unknown_ready_field") {
			t.Errorf("RawJSON = %s", issues[0].RawJSON)
		}
	})

	t.Run("list envelope omitted priority malformed timestamp", func(t *testing.T) {
		t.Parallel()
		issues, err := decodeIssues("List", []byte(`{
			"schema_version":1,
			"data":[{"id":"list-1","status":"closed","created_at":"not-a-time"}]
		}`))
		if err != nil {
			t.Fatalf("decodeIssues: %v", err)
		}
		if issues[0].Priority != nil {
			t.Errorf("Priority = %v, want nil", *issues[0].Priority)
		}
		if !issues[0].CreatedAt.IsZero() {
			t.Errorf("CreatedAt = %v, want zero", issues[0].CreatedAt)
		}
	})

	t.Run("show object and show dependency shape", func(t *testing.T) {
		t.Parallel()
		issue, ok, err := decodeOneIssue([]byte(`{
			"id":"show-1",
			"status":"open",
			"dependencies":[{"id":"parent-1","dependency_type":"blocks"}]
		}`))
		if err != nil {
			t.Fatalf("decodeOneIssue: %v", err)
		}
		if !ok {
			t.Fatal("ok = false")
		}
		if got := issue.Dependencies[0].ID; got != "parent-1" {
			t.Errorf("dependency ID = %q", got)
		}
	})
}

func TestIssueDecodeContractBadPayloads(t *testing.T) {
	t.Parallel()

	cases := map[string][]byte{
		"unsupported envelope": []byte(`{"schema_version":99,"data":[]}`),
		"missing id":           []byte(`[{"status":"open"}]`),
		"missing status":       []byte(`[{"id":"x"}]`),
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeIssues("Ready", payload)
			if !errors.Is(err, &Error{Kind: KindBadResponse}) {
				t.Fatalf("decodeIssues error = %v, want KindBadResponse", err)
			}
		})
	}
}

func TestIssueDecodeContractWrongShowID(t *testing.T) {
	t.Parallel()

	rr := newReplayRunner(t, replayRun{
		args:   []string{"--json", "show", "--", "want"},
		stdout: []byte(`[{"id":"got","status":"open"}]`),
	})
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.transport = &execTransport{config: c.config, runner: rr}

	_, err = c.Show(t.Context(), "want")
	if !errors.Is(err, &Error{Kind: KindBadResponse}) {
		t.Fatalf("Show error = %v, want KindBadResponse", err)
	}
	rr.assertDone()
}
