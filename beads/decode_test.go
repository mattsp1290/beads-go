package beads

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeIssuesLegacyArray(t *testing.T) {
	t.Parallel()

	got, err := decodeIssues("Ready", []byte(`[
		{"id":"a","status":"open","unknown":"raw"},
		{"id":"b","status":"closed"}
	]`))
	if err != nil {
		t.Fatalf("decodeIssues: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("issues = %+v", got)
	}
	if !strings.Contains(string(got[0].RawJSON), `"unknown":"raw"`) {
		t.Errorf("RawJSON = %s", got[0].RawJSON)
	}
}

func TestDecodeIssuesLegacyObject(t *testing.T) {
	t.Parallel()

	got, err := decodeIssues("Show", []byte(`{"id":"a","status":"open"}`))
	if err != nil {
		t.Fatalf("decodeIssues: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("got %+v", got)
	}
}

func TestDecodeIssuesEnvelopeArray(t *testing.T) {
	t.Parallel()

	got, err := decodeIssues("Ready", []byte(`{"schema_version":1,"data":[{"id":"a","status":"open"}]}`))
	if err != nil {
		t.Fatalf("decodeIssues: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("got %+v", got)
	}
}

func TestDecodeIssuesEnvelopeObject(t *testing.T) {
	t.Parallel()

	issue, ok, err := decodeOneIssue([]byte(`{"schema_version":1,"data":{"id":"a","status":"open"}}`))
	if err != nil {
		t.Fatalf("decodeOneIssue: %v", err)
	}
	if !ok || issue.ID != "a" {
		t.Fatalf("issue = %+v ok=%v", issue, ok)
	}
}

func TestDecodeIssuesEmptyPayloads(t *testing.T) {
	t.Parallel()

	for _, payload := range [][]byte{nil, []byte(""), []byte(" \n"), []byte("null"), []byte(`{"schema_version":1,"data":null}`)} {
		got, err := decodeIssues("Ready", payload)
		if err != nil {
			t.Fatalf("decodeIssues(%q): %v", payload, err)
		}
		if len(got) != 0 {
			t.Fatalf("decodeIssues(%q) len = %d", payload, len(got))
		}
	}
}

func TestDecodeIssuesBadPayloads(t *testing.T) {
	t.Parallel()

	for _, payload := range [][]byte{
		[]byte(`{"schema_version":2,"data":[]}`),
		[]byte(`{"schema_version":1}`),
		[]byte(`{"schema_version":1,"data":`),
		[]byte(`"string"`),
		[]byte(`[{"priority":"high"}]`),
	} {
		_, err := decodeIssues("Ready", payload)
		if !errors.Is(err, &Error{Kind: KindBadResponse}) {
			t.Fatalf("decodeIssues(%q) error = %v, want KindBadResponse", payload, err)
		}
	}
}

func TestDecodeOneIssueEmpty(t *testing.T) {
	t.Parallel()

	_, ok, err := decodeOneIssue([]byte(`[]`))
	if err != nil {
		t.Fatalf("decodeOneIssue: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false")
	}
}
