package beads

import (
	"bytes"
	"encoding/json"
	"time"
)

// Issue is the normalized issue shape returned by bd JSON commands.
type Issue struct {
	ID           string
	Title        string
	Description  string
	Status       string
	Priority     *int
	Labels       []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     time.Time
	Dependencies []Dependency
	RawJSON      json.RawMessage
}

// Dependency preserves bd dependency JSON fields from ready/list and show.
type Dependency struct {
	IssueID        string `json:"issue_id,omitempty"`
	DependsOnID    string `json:"depends_on_id,omitempty"`
	Type           string `json:"type,omitempty"`
	ID             string `json:"id,omitempty"`
	DependencyType string `json:"dependency_type,omitempty"`
}

// UnmarshalJSON decodes a bd issue object while preserving the original object
// in RawJSON. Unknown fields are intentionally ignored by typed fields.
func (i *Issue) UnmarshalJSON(data []byte) error {
	type wireIssue struct {
		ID           string       `json:"id"`
		Title        string       `json:"title"`
		Description  string       `json:"description"`
		Status       string       `json:"status"`
		Priority     *int         `json:"priority"`
		Labels       []string     `json:"labels"`
		CreatedAt    string       `json:"created_at"`
		UpdatedAt    string       `json:"updated_at"`
		ClosedAt     string       `json:"closed_at"`
		Dependencies []Dependency `json:"dependencies"`
	}

	var wire wireIssue
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	*i = Issue{
		ID:           wire.ID,
		Title:        wire.Title,
		Description:  wire.Description,
		Status:       wire.Status,
		Priority:     cloneIntPtr(wire.Priority),
		Labels:       append([]string(nil), wire.Labels...),
		CreatedAt:    parseBDTime(wire.CreatedAt),
		UpdatedAt:    parseBDTime(wire.UpdatedAt),
		ClosedAt:     parseBDTime(wire.ClosedAt),
		Dependencies: append([]Dependency(nil), wire.Dependencies...),
		RawJSON:      append(json.RawMessage(nil), bytes.TrimSpace(data)...),
	}
	return nil
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func parseBDTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
