package beads

import (
	"context"
	"strings"
)

// Transition moves an issue to state using bd update --status.
func (c *Client) Transition(ctx context.Context, id, state string) error {
	const op = "Transition"
	if c == nil {
		return validationError(op, "client is nil")
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return validationError(op, "state is required")
	}
	args, err := appendPositionalID(op, []string{"update", "--status=" + state}, id)
	if err != nil {
		return err
	}
	out, err := c.transport.execute(ctx, bdCall{op: op, args: args})
	if err != nil {
		return classifyExecError(op, err, out.stderr, out.stdout)
	}
	return nil
}
