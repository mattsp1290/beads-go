# ADR 0001: v1 scope and flat Client API

## Status

Accepted

## Context

`beads-go` starts as a small Go SDK for the bd CLI. The first real consumer is
local-symphony, but the SDK should remain reusable for other projects that need
programmatic access to bd.

## Decision

v0.1.0 exposes one public package:

```text
github.com/mattsp1290/beads-go/beads
```

The package exposes a concrete `Client`, not a public interface. Interfaces are
consumer-owned because each consumer has different workflow concepts and error
mapping needs.

`Client` is constructed with functional options:

- `WithBinary`
- `WithDataDir`
- `WithActor`
- `WithListLimit`

`Ready` and `List` use option-based filters:

- `WithState`
- `WithLabel`
- `WithAll`

Filters are options rather than positional parameters so future filters can be
added without changing method signatures.

The v0.1.0 method surface is flat and limited to:

- `Ready`
- `List`
- `Show`
- `Close`
- `Comment`
- `Transition`

## Non-Goals

The SDK does not export:

- local-symphony's `TrackerReader`, `TrackerWriter`, or `Tracker`
- a public transport interface
- a public runner interface
- operation enums
- request envelopes
- `Raw(args ...string)` or any other CLI-shaped escape hatch

The following bd surfaces are deferred:

- `Create`
- broader `Update`
- `Dep` add/remove/tree
- `Graph`
- `Init`
- `Remember`
- `Prime`
- `Dolt`
- `RateLimit`

Promotion trigger: add these only when a Go program needs them directly and the
method-shaped API can be designed without exposing raw CLI argv.

## Consequences

The API is intentionally small and easy to wrap. local-symphony can adapt
`*beads.Client` into its local tracker interfaces without forcing those
interfaces into this module.

A future RPC transport can fit behind the current client methods without
emulating arbitrary CLI commands.

Pre-v1 breaking changes remain possible, but every breaking change needs a
changelog migration note.
