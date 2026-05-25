package beads

import (
	"context"
	"strconv"
)

// Ready returns bd issues that are ready to work.
func (c *Client) Ready(ctx context.Context, opts ...ListOption) ([]Issue, error) {
	const op = "Ready"
	if c == nil {
		return nil, validationError(op, "client is nil")
	}
	cfg, err := applyListOptions(opts)
	if err != nil {
		return nil, err
	}

	args := []string{"ready"}
	for _, label := range cfg.labels {
		args = append(args, "--label="+label)
	}
	switch {
	case cfg.all:
		args = append(args, "-n", "0")
	case c.config.listLimit > 0:
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
	return filterIssuesByState(issues, cfg.states), nil
}

func filterIssuesByState(issues []Issue, states []string) []Issue {
	out := make([]Issue, 0, len(issues))
	if len(states) == 0 {
		return append(out, issues...)
	}
	allowed := make(map[string]struct{}, len(states))
	for _, state := range states {
		allowed[state] = struct{}{}
	}
	for _, issue := range issues {
		if _, ok := allowed[issue.Status]; ok {
			out = append(out, issue)
		}
	}
	return out
}
