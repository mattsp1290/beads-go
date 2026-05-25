package beads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func decodeIssues(op string, payload []byte) ([]Issue, error) {
	raw, err := unwrapPayload(payload)
	if err != nil {
		return nil, badResponseError(op, err)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	switch raw[0] {
	case '[':
		var issues []Issue
		if err := json.Unmarshal(raw, &issues); err != nil {
			return nil, badResponseError(op, err)
		}
		if err := validateIssues(op, issues); err != nil {
			return nil, err
		}
		return issues, nil
	case '{':
		var issue Issue
		if err := json.Unmarshal(raw, &issue); err != nil {
			return nil, badResponseError(op, err)
		}
		if err := validateIssues(op, []Issue{issue}); err != nil {
			return nil, err
		}
		return []Issue{issue}, nil
	default:
		return nil, badResponseError(op, fmt.Errorf("unexpected JSON payload %q", raw[0]))
	}
}

func validateIssues(op string, issues []Issue) error {
	for idx, issue := range issues {
		if strings.TrimSpace(issue.ID) == "" {
			return badResponseError(op, fmt.Errorf("issue at index %d missing required id", idx))
		}
		if strings.TrimSpace(issue.Status) == "" {
			return badResponseError(op, fmt.Errorf("issue %q missing required status", issue.ID))
		}
	}
	return nil
}

func decodeOneIssue(payload []byte) (Issue, bool, error) {
	issues, err := decodeIssues("Show", payload)
	if err != nil {
		return Issue{}, false, err
	}
	if len(issues) == 0 {
		return Issue{}, false, nil
	}
	return issues[0], true, nil
}

func unwrapPayload(payload []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	if trimmed[0] != '{' {
		return trimmed, nil
	}

	var probe struct {
		SchemaVersion *int            `json:"schema_version"`
		Data          json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return nil, err
	}
	if probe.SchemaVersion == nil {
		return trimmed, nil
	}
	if *probe.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported bd JSON envelope schema_version %d", *probe.SchemaVersion)
	}
	if len(bytes.TrimSpace(probe.Data)) == 0 {
		return nil, fmt.Errorf("bd JSON envelope missing data")
	}
	if bytes.Equal(bytes.TrimSpace(probe.Data), []byte("null")) {
		return nil, nil
	}
	return bytes.TrimSpace(probe.Data), nil
}
