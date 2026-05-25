package beads

import (
	"context"
	"errors"
	"testing"
)

func TestIDTakingMethodsRejectInvalidIDsBeforeRunner(t *testing.T) {
	t.Parallel()

	methods := []struct {
		name string
		call func(*Client, string) error
	}{
		{
			name: "Show",
			call: func(c *Client, id string) error {
				_, err := c.Show(context.Background(), id)
				return err
			},
		},
		{
			name: "Close",
			call: func(c *Client, id string) error {
				return c.Close(context.Background(), id, "done")
			},
		},
		{
			name: "Comment",
			call: func(c *Client, id string) error {
				return c.Comment(context.Background(), id, "body")
			},
		},
		{
			name: "Transition",
			call: func(c *Client, id string) error {
				return c.Transition(context.Background(), id, "closed")
			},
		},
	}

	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			t.Parallel()
			for _, id := range []string{"", " \t\n", "-rf"} {
				t.Run(id, func(t *testing.T) {
					t.Parallel()
					c, err := NewClient()
					if err != nil {
						t.Fatalf("NewClient: %v", err)
					}
					c.transport = &execTransport{config: c.config, runner: newPanicRunner(t)}
					if err := method.call(c, id); !errors.Is(err, ErrValidation) {
						t.Fatalf("%s(%q) error = %v, want ErrValidation", method.name, id, err)
					}
				})
			}
		})
	}
}
