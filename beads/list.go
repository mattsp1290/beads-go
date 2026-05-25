package beads

import (
	"context"
	"strconv"
	"strings"
)

// List returns issues matching the supplied filters.
func (c *Client) List(ctx context.Context, opts ...ListOption) ([]Issue, error) {
	const op = "List"
	if c == nil {
		return nil, validationError(op, "client is nil")
	}
	cfg, err := applyListOptions(opts)
	if err != nil {
		return nil, err
	}

	args := []string{"list"}
	if cfg.all {
		args = append(args, "--all")
	}
	if len(cfg.states) > 0 {
		args = append(args, "--status="+strings.Join(cfg.states, ","))
	}
	for _, label := range cfg.labels {
		args = append(args, "--label="+label)
	}
	if c.config.listLimit > 0 {
		args = append(args, "-n", strconv.Itoa(c.config.listLimit))
	}

	out, err := c.transport.execute(ctx, bdCall{op: op, args: args})
	if err != nil {
		return nil, classifyExecError(op, err, out.stderr, out.stdout)
	}
	issues, err := decodeIssues(op, out.stdout)
	if err != nil {
		return nil, err
	}
	return filterIssuesByState(issues, nil), nil
}
