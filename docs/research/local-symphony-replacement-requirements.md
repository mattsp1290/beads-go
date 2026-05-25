# local-symphony replacement requirements

Date: 2026-05-25

This note narrows the earlier bd command audit into the exact work needed to
replace `~/git/local-symphony/internal/tracker/beads` with a wrapper around
`github.com/mattsp1290/beads-go/beads`.

## Runtime wiring to preserve

`local-symphony/internal/runtime/runtime.go` imports:

```go
beadsadapter "github.com/mattsp1290/local-symphony/internal/tracker/beads"
```

The runtime factory currently calls:

```go
r.Tracker = fac.newTracker(beadsadapter.Config{
	DataDir:     wf.Config.Tracker.DataDir,
	CallTimeout: time.Duration(wf.Config.Tracker.CallTimeoutMs) * time.Millisecond,
	Logger:      log,
})
```

The lowest-churn replacement is to keep the local package path
`internal/tracker/beads` and keep a local `Config` plus `New(Config)` function.
That local package should construct a `*beads.Client` with:

- `beads.NewClient`
- `beads.WithBinary` when local `Config.Binary` is non-empty
- `beads.WithDataDir` from local `Config.DataDir`
- `beads.WithActor` from local `Config.Actor`
- `beads.WithListLimit` from local `Config.ListLimit`

`CallTimeout` and `Logger` remain local-symphony wrapper concerns because the
SDK intentionally uses context-only timeouts and has no logging/telemetry.
Wrapper methods should derive per-call contexts when `CallTimeout > 0` and log
the same timeout warning fields local-symphony expects today.

## Wrapper method behavior

The local wrapper must continue satisfying:

- `tracker.TrackerReader`
- `tracker.TrackerWriter`
- `tracker.Tracker`

Reader behavior:

- `FetchCandidates(ctx, activeStates)` must call `Client.Ready` even when
  `activeStates` is empty, then return a non-nil empty slice for an empty active
  set. For non-empty active sets, it may pass `beads.WithState` filters or
  filter after decode, as long as behavior matches existing tests.
- `FetchByStates(ctx, states)` must short-circuit nil/empty/all-empty states
  without invoking bd. For non-empty states, preserve the existing per-state
  fan-out if tests still assert call count; otherwise `Client.List` supports a
  comma-joined state filter. Results must be deduped by issue ID.
- `FetchStatesByIDs(ctx, ids)` must return an empty map without bd calls for
  nil/empty input, skip empty IDs, dedupe repeated IDs, call `Client.Show` once
  per unique ID, omit not-found IDs, and return a partial map plus error on hard
  failures.
- `RateLimit()` returns the zero `tracker.RateLimitSnapshot`.

Issue mapping behavior:

- `beads.Issue.ID` maps to `core.Issue.ID`.
- `core.Issue.Identifier` falls back to ID because beads-go does not expose a
  separate identifier field.
- `Status` maps through as `core.IssueState`.
- `Priority == nil` maps to `core.PriorityUnset`; 0..4 map to critical, high,
  medium, low, backlog; out-of-range maps unset.
- Labels must be lowercased, trimmed, deduped, and sorted in the wrapper.
- Dependencies with kind `blocks` map to `core.Issue.BlockedBy`; preserve both
  bd shapes (`depends_on_id/type` and `id/dependency_type`), drop self
  references, dedupe, and sort.
- Timestamps from beads-go are already UTC or zero; the wrapper should pass
  them through.

Writer behavior:

- `Comment` calls `Client.Comment`; non-idempotent retry dedupe remains above
  the tracker layer.
- `Transition` calls `Client.Transition`.
- `Close` calls `Client.Close`, preserving trimmed reason and omission of empty
  reason.
- `LinkPR` remains locally unsupported and must map to local
  `tracker.ErrUnsupported` / `CategoryUnsupported`.

Error mapping:

- `beads.KindBadResponse` -> `tracker.CategoryUnknownPayload`
- `beads.KindAuthFailed` -> `tracker.CategoryAuthFailed`
- `beads.KindExit` -> `tracker.CategoryAPIStatus` with status preserved
- `beads.KindValidation` -> `tracker.CategoryValidation`
- `beads.KindTimeout` -> `tracker.CategoryTimeout`
- `beads.KindTransport` -> `tracker.CategoryAPIRequest`
- `beads.KindNotFound` -> `tracker.CategoryNotFound`
- `beads.KindUnsupported` -> `tracker.CategoryUnsupported`

The local wrapper should preserve `errors.Is` behavior for unsupported and
not-found where existing tests expect it.

## Tests that must remain behavior-compatible

Keep or update these local-symphony tests around the wrapper:

- `internal/tracker/beads/beads_test.go`: reader fan-out, writer argv behavior
  translated to SDK calls, RateLimit zero, LinkPR unsupported, timeout handling.
- `internal/tracker/beads/parse_test.go`: replace parser-only assertions with
  wrapper mapping tests against `beads.Issue` fixtures.
- `internal/tracker/beads/errors_test.go`: SDK kind to local tracker category
  mapping, status preservation, sentinels.
- `internal/tracker/tracker_test.go` and `internal/tracker/errors_test.go`:
  interface and category contracts.
- `internal/runtime/runtime_test.go`: `TestBuildAppliesTrackerDataDirFromWorkflow`
  must still prove workflow tracker data_dir flows into the local beads wrapper.
- `internal/worker/tools/trackerwrite/trackerwrite_test.go`: `op=close` still
  reaches `TrackerWriter.Close`; `comment`, `transition`, and `link_pr` remain
  `unsupported_op` at the tool layer in local-symphony v1.

Consumer acceptance should also run the relevant orchestrator/reconcile tests
that exercise `TrackerReader.FetchCandidates`, `FetchByStates`, and
`FetchStatesByIDs` through fakes, plus the broader local-symphony suite selected
by the replacement PR.
