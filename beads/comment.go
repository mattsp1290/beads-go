package beads

import (
	"context"
	"strings"
)

// Comment appends body to an issue's bd notes.
func (c *Client) Comment(ctx context.Context, id, body string) error {
	const op = "Comment"
	if c == nil {
		return validationError(op, "client is nil")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return validationError(op, "comment body is required")
	}
	args, err := appendPositionalID(op, []string{"update", "--append-notes=" + body}, id)
	if err != nil {
		return err
	}
	out, err := c.transport.execute(ctx, bdCall{op: op, args: args})
	if err != nil {
		return classifyExecError(op, err, out.stderr, out.stdout)
	}
	return nil
}
