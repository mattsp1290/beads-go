package beads

import "strings"

const defaultBinary = "bd"

// Option configures a Client at construction time.
type Option func(*clientConfig) error

// Client wraps bd command execution.
//
// A Client is safe for concurrent use by multiple goroutines. Its configuration
// is copied during construction and is not mutated after NewClient returns.
type Client struct {
	config    clientConfig
	transport transport
}

type clientConfig struct {
	binary    string
	dataDir   string
	actor     string
	listLimit int
}

// NewClient constructs a Client using bd from PATH unless WithBinary is used.
func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{binary: defaultBinary}
	for i, opt := range opts {
		if opt == nil {
			return nil, validationError("NewClient", "option %d is nil", i)
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	return &Client{config: cfg, transport: newExecTransport(cfg)}, nil
}

// WithBinary sets the bd executable name or absolute path. Empty uses "bd"
// from PATH.
func WithBinary(binary string) Option {
	return func(cfg *clientConfig) error {
		if strings.TrimSpace(binary) == "" {
			cfg.binary = defaultBinary
			return nil
		}
		cfg.binary = binary
		return nil
	}
}

// WithDataDir sets the bd data directory passed as --db=<dir>. Empty leaves bd
// to auto-discover its data directory.
func WithDataDir(dir string) Option {
	return func(cfg *clientConfig) error {
		cfg.dataDir = dir
		return nil
	}
}

// WithActor sets the bd audit actor passed as --actor=<name>. Empty omits the
// actor flag.
func WithActor(actor string) Option {
	return func(cfg *clientConfig) error {
		cfg.actor = actor
		return nil
	}
}

// WithListLimit sets the default limit used by list-like calls. Zero leaves bd
// defaults in place.
func WithListLimit(limit int) Option {
	return func(cfg *clientConfig) error {
		if limit < 0 {
			return validationError("NewClient", "list limit must be non-negative")
		}
		cfg.listLimit = limit
		return nil
	}
}
