package beads

import (
	"context"
	"strings"
)

// Close closes an issue, optionally recording reason.
func (c *Client) Close(ctx context.Context, id, reason string) error {
	const op = "Close"
	if c == nil {
		return validationError(op, "client is nil")
	}
	args := []string{"close"}
	if reason = strings.TrimSpace(reason); reason != "" {
		args = append(args, "--reason="+reason)
	}
	args, err := appendPositionalID(op, args, id)
	if err != nil {
		return err
	}
	out, err := c.transport.execute(ctx, bdCall{op: op, args: args})
	if err != nil {
		return classifyExecError(op, err, out.stderr, out.stdout)
	}
	return nil
}
