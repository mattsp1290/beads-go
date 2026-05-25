// Package beads provides a small Go SDK for the bd command-line issue tracker.
//
// The package shells out to the operator-configured bd binary and exposes the
// v1 read/write surface needed by agentic projects: ready work, filtered lists,
// issue lookup, close, comment, and transition. It intentionally does not expose
// a raw argv escape hatch, public transport seam, or runner package.
//
// A Client is safe for concurrent use by multiple goroutines. The client holds
// immutable configuration and per-call state only. Concurrency with other bd
// processes is still governed by bd itself: the default embedded Dolt database
// holds an exclusive lock on the .beads data directory while a command is using
// it. Applications that need multi-writer behavior should use bd's server-backed
// mode or serialize access at a higher layer.
//
// All operations take a context.Context. This package does not add its own
// timeout, retry, logging, metrics, or tracing policy; callers own those
// concerns by deriving contexts and observing returned errors.
//
// JSON decoding is forward-compatible with bd schema additions. Unknown fields
// are ignored for typed access while each Issue keeps its original RawJSON so
// callers can inspect newly added bd fields before this package grows first-class
// support for them. Missing required fields and malformed JSON are reported as
// bad-response errors rather than being silently defaulted.
//
// This module is pre-v1. Breaking changes may occur under v0.x and must be
// documented with migration notes in CHANGELOG.md.
package beads
