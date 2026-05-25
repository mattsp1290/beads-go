package beads

import (
	"errors"
	"slices"
	"testing"
)

func TestApplyListOptions(t *testing.T) {
	t.Parallel()

	cfg, err := applyListOptions([]ListOption{
		WithState(" open "),
		WithState("closed"),
		WithLabel(" setup "),
		WithLabel("api"),
		WithAll(),
	})
	if err != nil {
		t.Fatalf("applyListOptions: %v", err)
	}
	if !slices.Equal(cfg.states, []string{"open", "closed"}) {
		t.Errorf("states = %v", cfg.states)
	}
	if !slices.Equal(cfg.labels, []string{"setup", "api"}) {
		t.Errorf("labels = %v", cfg.labels)
	}
	if !cfg.all {
		t.Error("all = false, want true")
	}
}

func TestApplyListOptionsValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts []ListOption
	}{
		{"nil option", []ListOption{nil}},
		{"empty state", []ListOption{WithState(" \t")}},
		{"empty label", []ListOption{WithLabel("")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := applyListOptions(tc.opts)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("applyListOptions error = %v, want ErrValidation", err)
			}
		})
	}
}
