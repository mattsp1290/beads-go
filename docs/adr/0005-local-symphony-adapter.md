# ADR 0005: local-symphony adapter contract

## Status

Accepted

## Context

`local-symphony` already has internal tracker interfaces:

- `TrackerReader`
- `TrackerWriter`
- `Tracker`

Those interfaces include local orchestration concepts such as
`core.IssueState`, `core.Priority`, `RateLimitSnapshot`, local error
categories, and a `LinkPR` method that beads itself cannot support. Exporting
that triad from `beads-go` would make this SDK a local-symphony extension
instead of a reusable bd client.

`beads-go` v0.1.0 therefore exposes a concrete `*beads.Client` with bd-shaped
methods and data:

- `Ready`
- `List`
- `Show`
- `Close`
- `Comment`
- `Transition`

`local-symphony` remains the first consumer, but it should adapt the SDK locally
rather than forcing its internal tracker abstraction into this module.

## Decision

`beads-go` will not export local-symphony's `TrackerReader`, `TrackerWriter`, or
`Tracker` interfaces.

`local-symphony/internal/tracker/beads` should remain as a thin wrapper package.
That wrapper should keep its local `Config` and `New(Config)` entry point so
runtime wiring can stay small, then construct a `*beads.Client` with
`beads.NewClient` and functional options.

The local wrapper owns:

- Mapping `beads.Issue` into `local-symphony/internal/core.Issue`.
- Mapping `beads.Error.Kind` into local `tracker.Category` values.
- `RateLimit()` returning the zero local snapshot.
- `LinkPR` returning local unsupported errors.
- Per-call timeout contexts and timeout logging.
- Any local tracker conformance tests and runtime wiring tests.

## Required Wrapper Behavior

`FetchCandidates(ctx, activeStates)`:

- Calls `Client.Ready` every time, including when `activeStates` is empty.
- Returns a non-nil empty slice for an empty active-state set.
- Filters candidates to requested active states.

`FetchByStates(ctx, states)`:

- Returns a non-nil empty slice without bd calls for nil/empty/all-empty input.
- Calls `Client.List` with `WithAll` and state filters for non-empty states.
- Preserves existing dedupe-by-ID behavior.

`FetchStatesByIDs(ctx, ids)`:

- Returns an empty map without bd calls for nil/empty input.
- Skips empty IDs and dedupes repeated IDs.
- Calls `Client.Show` once per unique ID.
- Omits not-found IDs.
- Returns partial results plus the first hard error.

Writer behavior:

- `Comment` delegates to `Client.Comment`.
- `Transition` delegates to `Client.Transition`.
- `Close` delegates to `Client.Close`.
- `LinkPR` remains unsupported locally.

Error mapping:

- `KindBadResponse` -> `CategoryUnknownPayload`
- `KindAuthFailed` -> `CategoryAuthFailed`
- `KindExit` -> `CategoryAPIStatus`
- `KindValidation` -> `CategoryValidation`
- `KindTimeout` -> `CategoryTimeout`
- `KindTransport` -> `CategoryAPIRequest`
- `KindNotFound` -> `CategoryNotFound`
- `KindUnsupported` -> `CategoryUnsupported`

## Consequences

`beads-go` stays a small bd SDK with no dependency on local-symphony internals.

The consumer wrapper is responsible for preserving local behavior, including
label normalization, priority mapping, blocked-by derivation, timeout logging,
and existing tracker error categories.

The local-symphony replacement PR must keep or update the current
`internal/tracker/beads`, `internal/runtime`, and `tracker_write` tests so they
prove the wrapper remains behavior-compatible.
