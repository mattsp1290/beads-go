package beads

import (
	"strings"
)

func requireID(op, id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", validationError(op, "issue id is required")
	}
	if strings.HasPrefix(trimmed, "-") {
		return "", validationError(op, "issue id %q must not start with '-'", trimmed)
	}
	return trimmed, nil
}

func appendPositionalID(op string, args []string, id string) ([]string, error) {
	validID, err := requireID(op, id)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(args)+2)
	out = append(out, args...)
	out = append(out, "--", validID)
	return out, nil
}
