package beads

import (
	"strings"
	"sync"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient returned nil client")
	}
	if c.config.binary != defaultBinary {
		t.Errorf("binary = %q, want %q", c.config.binary, defaultBinary)
	}
	if c.config.dataDir != "" {
		t.Errorf("dataDir = %q, want empty", c.config.dataDir)
	}
	if c.config.actor != "" {
		t.Errorf("actor = %q, want empty", c.config.actor)
	}
	if c.config.listLimit != 0 {
		t.Errorf("listLimit = %d, want 0", c.config.listLimit)
	}
}

func TestNewClientAppliesOptions(t *testing.T) {
	t.Parallel()

	c, err := NewClient(
		WithBinary("/usr/local/bin/bd"),
		WithDataDir("/var/lib/beads/project"),
		WithActor("orchestrator"),
		WithListLimit(50),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.config.binary != "/usr/local/bin/bd" {
		t.Errorf("binary = %q", c.config.binary)
	}
	if c.config.dataDir != "/var/lib/beads/project" {
		t.Errorf("dataDir = %q", c.config.dataDir)
	}
	if c.config.actor != "orchestrator" {
		t.Errorf("actor = %q", c.config.actor)
	}
	if c.config.listLimit != 50 {
		t.Errorf("listLimit = %d", c.config.listLimit)
	}
}

func TestNewClientRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(nil); err == nil {
		t.Fatal("NewClient(nil): expected error")
	}
	if _, err := NewClient(WithListLimit(-1)); err == nil {
		t.Fatal("NewClient(WithListLimit(-1)): expected error")
	}
}

func TestWithBinaryEmptyUsesDefault(t *testing.T) {
	t.Parallel()

	for _, binary := range []string{"", " \t\n"} {
		t.Run(strings.ReplaceAll(binary, "\n", `\n`), func(t *testing.T) {
			t.Parallel()
			c, err := NewClient(WithBinary(binary))
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if c.config.binary != defaultBinary {
				t.Errorf("binary = %q, want %q", c.config.binary, defaultBinary)
			}
		})
	}
}

func TestClientConfigIsStableAcrossConcurrentReads(t *testing.T) {
	t.Parallel()

	c, err := NewClient(
		WithBinary("/bin/bd"),
		WithDataDir("/state"),
		WithActor("actor"),
		WithListLimit(25),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			if c.config.binary != "/bin/bd" ||
				c.config.dataDir != "/state" ||
				c.config.actor != "actor" ||
				c.config.listLimit != 25 {
				t.Errorf("config mutated: %+v", c.config)
			}
		}()
	}
	wg.Wait()
}
