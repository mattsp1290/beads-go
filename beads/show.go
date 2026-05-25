package beads

import (
	"context"
	"fmt"
)

// Show fetches one issue by ID.
func (c *Client) Show(ctx context.Context, id string) (Issue, error) {
	const op = "Show"
	if c == nil {
		return Issue{}, validationError(op, "client is nil")
	}
	validID, err := requireID(op, id)
	if err != nil {
		return Issue{}, err
	}
	args := []string{"show", "--", validID}
	out, err := c.transport.execute(ctx, bdCall{op: op, args: args})
	if err != nil {
		return Issue{}, classifyExecError(op, err, out.stderr, out.stdout)
	}
	issue, ok, err := decodeOneIssue(out.stdout)
	if err != nil {
		return Issue{}, err
	}
	if !ok {
		return Issue{}, &Error{Op: op, Kind: KindNotFound, Err: fmt.Errorf("issue %q not found", validID)}
	}
	if issue.ID == "" {
		return Issue{}, badResponseError(op, fmt.Errorf("bd show %q returned issue with empty id", validID))
	}
	if issue.ID != validID {
		return Issue{}, badResponseError(op, fmt.Errorf("bd show %q returned id %q", validID, issue.ID))
	}
	return issue, nil
}
