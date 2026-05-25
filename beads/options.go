package beads

import "strings"

// ListOption configures Ready and List calls.
type ListOption func(*listConfig) error

type listConfig struct {
	states []string
	labels []string
	all    bool
}

// WithState filters list-like calls by status/state.
func WithState(state string) ListOption {
	return func(cfg *listConfig) error {
		state = strings.TrimSpace(state)
		if state == "" {
			return validationError("ListOption", "state is required")
		}
		cfg.states = append(cfg.states, state)
		return nil
	}
}

// WithLabel filters list-like calls by label.
func WithLabel(label string) ListOption {
	return func(cfg *listConfig) error {
		label = strings.TrimSpace(label)
		if label == "" {
			return validationError("ListOption", "label is required")
		}
		cfg.labels = append(cfg.labels, label)
		return nil
	}
}

// WithAll requests all matching issues instead of bd's default page.
func WithAll() ListOption {
	return func(cfg *listConfig) error {
		cfg.all = true
		return nil
	}
}

func applyListOptions(opts []ListOption) (listConfig, error) {
	var cfg listConfig
	for i, opt := range opts {
		if opt == nil {
			return listConfig{}, validationError("ListOption", "option %d is nil", i)
		}
		if err := opt(&cfg); err != nil {
			return listConfig{}, err
		}
	}
	return cfg, nil
}
