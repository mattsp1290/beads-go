package beads

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestIssueUnmarshalNormalizesFieldsAndPreservesRawJSON(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"id":"beadsgo-1",
		"title":"Title",
		"description":"Desc",
		"status":"open",
		"priority":0,
		"labels":["Setup","API"],
		"created_at":"2026-05-25T12:00:00-07:00",
		"updated_at":"2026-05-25T20:01:02Z",
		"closed_at":"2026-05-26T03:04:05+02:00",
		"extra_field":"kept in raw",
		"dependencies":[
			{"issue_id":"beadsgo-1","depends_on_id":"beadsgo-0","type":"blocks"},
			{"id":"beadsgo-2","dependency_type":"related"}
		]
	}`)

	var issue Issue
	if err := json.Unmarshal(payload, &issue); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if issue.ID != "beadsgo-1" || issue.Title != "Title" || issue.Description != "Desc" || issue.Status != "open" {
		t.Errorf("issue fields = %+v", issue)
	}
	if issue.Priority == nil || *issue.Priority != 0 {
		t.Fatalf("Priority = %v, want pointer to 0", issue.Priority)
	}
	if !slices.Equal(issue.Labels, []string{"Setup", "API"}) {
		t.Errorf("Labels = %v", issue.Labels)
	}
	if got, want := issue.CreatedAt, mustParseTime("2026-05-25T19:00:00Z"); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("CreatedAt = %v, want %v in UTC", got, want)
	}
	if got, want := issue.UpdatedAt, mustParseTime("2026-05-25T20:01:02Z"); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("UpdatedAt = %v, want %v in UTC", got, want)
	}
	if got, want := issue.ClosedAt, mustParseTime("2026-05-26T01:04:05Z"); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("ClosedAt = %v, want %v in UTC", got, want)
	}
	if len(issue.Dependencies) != 2 {
		t.Fatalf("Dependencies len = %d", len(issue.Dependencies))
	}
	if issue.Dependencies[0].IssueID != "beadsgo-1" ||
		issue.Dependencies[0].DependsOnID != "beadsgo-0" ||
		issue.Dependencies[0].Type != "blocks" {
		t.Errorf("ready/list dependency shape = %+v", issue.Dependencies[0])
	}
	if issue.Dependencies[1].ID != "beadsgo-2" ||
		issue.Dependencies[1].DependencyType != "related" {
		t.Errorf("show dependency shape = %+v", issue.Dependencies[1])
	}
	if !strings.Contains(string(issue.RawJSON), `"extra_field":"kept in raw"`) {
		t.Errorf("RawJSON did not preserve unknown field: %s", issue.RawJSON)
	}
}

func TestIssueUnmarshalMissingAndMalformedTimesAreZero(t *testing.T) {
	t.Parallel()

	var issue Issue
	if err := json.Unmarshal([]byte(`{"id":"x","status":"open","created_at":"bad"}`), &issue); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !issue.CreatedAt.IsZero() {
		t.Errorf("CreatedAt = %v, want zero", issue.CreatedAt)
	}
	if !issue.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt = %v, want zero", issue.UpdatedAt)
	}
	if !issue.ClosedAt.IsZero() {
		t.Errorf("ClosedAt = %v, want zero", issue.ClosedAt)
	}
}

func TestIssueUnmarshalPriorityAbsent(t *testing.T) {
	t.Parallel()

	var issue Issue
	if err := json.Unmarshal([]byte(`{"id":"x","status":"open"}`), &issue); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if issue.Priority != nil {
		t.Errorf("Priority = %v, want nil", *issue.Priority)
	}
}

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
