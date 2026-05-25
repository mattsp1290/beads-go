package beads

import (
	"slices"
	"testing"
)

func TestRequireID(t *testing.T) {
	t.Parallel()

	t.Run("valid trimmed", func(t *testing.T) {
		t.Parallel()
		got, err := requireID("Show", "  beadsgo-123  ")
		if err != nil {
			t.Fatalf("requireID: %v", err)
		}
		if got != "beadsgo-123" {
			t.Errorf("id = %q, want trimmed id", got)
		}
	})

	for _, id := range []string{"", " ", "\t\n"} {
		t.Run("empty", func(t *testing.T) {
			t.Parallel()
			if _, err := requireID("Show", id); err == nil {
				t.Fatalf("requireID(%q): expected error", id)
			}
		})
	}

	for _, id := range []string{"-x", " --help "} {
		t.Run("dash prefix", func(t *testing.T) {
			t.Parallel()
			if _, err := requireID("Show", id); err == nil {
				t.Fatalf("requireID(%q): expected error", id)
			}
		})
	}
}

func TestAppendPositionalIDAddsSeparator(t *testing.T) {
	t.Parallel()

	base := []string{"close", "--reason=done"}
	got, err := appendPositionalID("Close", base, "issue-1")
	if err != nil {
		t.Fatalf("appendPositionalID: %v", err)
	}
	want := []string{"close", "--reason=done", "--", "issue-1"}
	if !slices.Equal(got, want) {
		t.Errorf("args = %v, want %v", got, want)
	}
	if !slices.Equal(base, []string{"close", "--reason=done"}) {
		t.Errorf("base mutated: %v", base)
	}
}

func TestAppendPositionalIDRejectsInvalidID(t *testing.T) {
	t.Parallel()

	if got, err := appendPositionalID("Close", []string{"close"}, "-bad"); err == nil {
		t.Fatalf("expected error, got args %v", got)
	}
}
