# Extract `beads-go`: planning document

Status: plan only. No Go source, no `go mod init`, no scaffolding.
Target module path (proposed): `github.com/mattsp1290/beads-go`.
Source-of-truth runs in this plan were performed on 2026-05-25 against
`bd version 1.0.0 (72170267)` installed at `/usr/local/bin/bd`.

The canonical CLI conventions reference is
`~/git/dotfiles/.agents/rules/beads.md`. That file covers a tiny slice of
the surface area, so most verb-level claims below cite `bd <subcommand>
--help` output captured on 2026-05-25 from a real install.

---

## 1. Goals and non-goals

### Goals

- A typed Go client over the `bd` CLI that exposes the task-graph
  operations both `advisor` and `local-symphony` need today: `ready`,
  `list`, `show`, `create`, `close`, `update` (including `--claim`),
  `delete`, `dep add`, `dep remove`, `dep tree`, `dep cycles`, `graph`,
  `prime`, `export`, and `dolt push` (the last gated behind an explicit
  caller call).
- A `TrackerWriter`-friendly subset matching the interface in
  `~/git/local-symphony/internal/tracker/tracker.go` (lines 58-76):
  `Comment`, `Transition`, `Close`, `LinkPR`. `LinkPR` returns
  `ErrUnsupported` (beads has no PR concept; see the existing adapter's
  rationale at `internal/tracker/beads/beads.go` line 348).
- A `TrackerReader` subset matching the same file (lines 18-50):
  `FetchCandidates`, `FetchByStates`, `FetchStatesByIDs`, `RateLimit`.
  `RateLimit()` returns the zero `RateLimitSnapshot` — there is no
  remote budget to surface for a local CLI.
- Compile-time interface conformance assertions for both
  `tracker.TrackerReader` and `tracker.TrackerWriter`, however that
  interface ends up being hosted (see §15 open question on interface
  boundary).
- Hermetic test seam: a `runner` interface so tests can substitute a
  fake without spawning `bd`. The pattern is already proven in
  `internal/tracker/beads/runner.go` and should be lifted verbatim.

### Non-goals

- An embedded beads server. `bd` stays a separate upstream project at
  `gastownhall/beads`; `beads-go` only wraps the CLI.
- TUI / GUI helpers. The terminal-DAG rendering of `bd graph` stays in
  `bd`; if a downstream caller wants graph visualization it captures
  the `--dot` output (see §6) and pipes it.
- LLM tool wrappers. Symphony's `tracker_write` tool and the planned
  `eino-tools/trackerwrite` belong on top of `beads-go`, not inside it.
- Daemonization or a long-lived client cache. Every method is one or
  more `bd` invocations; concurrency control is the caller's problem
  (with help — see §8).
- Coverage of `bd`'s entire 60-plus-verb surface. v0.1 covers what
  advisor and symphony need; later versions add the rest. See §11.

---

## 2. Why this is worth doing now

The standard "extract on the third consumer" rule is borderline here.
Two existing in-tree consumers and a designed-for-it interface push
this over the line:

- **Two repositories already mandate `bd` via agent rules.**
  - `~/git/advisor/CLAUDE.md` lines 7-50 ("Beads Issue Tracker" /
    "Session Completion") instruct every agent in the repo to use `bd
    ready`, `bd update --claim`, `bd close`, `bd dolt push`.
  - `~/git/local-symphony/CLAUDE.md` has the same `bd prime` /
    `bd update --claim` mandate.
- **Symphony's `tracker.TrackerWriter` interface is shaped for beads
  as the first adapter** (`internal/tracker/tracker.go` line 58
  onward), and the adapter at `internal/tracker/beads/` is already
  600-plus lines of production code plus 1000-plus lines of tests
  (see `beads.go`, `runner.go`, `parse.go`, `errors.go` and the
  matching `_test.go` files). That code is the v0.1 of `beads-go`
  hiding inside symphony's `internal/` tree.
- **`advisor` does NOT currently shell out to `bd` from Go.** Greps
  for `exec.Command.*bd` across `~/git/advisor/internal` and
  `~/git/advisor/cmd` come back empty; the only `bd` references are
  in `CLAUDE.md`, `AGENTS.md`, and the `setup-beads.sh` bootstrap.
  This is the most important deviation from the original framing:
  advisor is a *future* consumer, not a deduplication target. The
  case for `beads-go` therefore rests on (a) symphony already having
  built the adapter and (b) every new advisor flow that wants to file
  follow-up issues or close session beads programmatically benefits
  from a shared client.

### Soft cost / pushback

- A Go client over a CLI couples to the CLI's stability. `bd` 1.0.0
  has a semver flag-stability promise — meaningfully better than the
  task's original "pre-1.0 churn" framing — but JSON-schema additions
  are not versioned. Mitigation in §13.
- `os/exec` adds ~ms per call. Fine for orchestrator-level work;
  tight loops should prefer `bd list` over fan-out `bd show`.
- Biggest hazard: `bd update --claim` racing across workspaces. See §8.

---

## 3. Transport choice

Two options were considered:

**(a) `os/exec` shell-out + `--json`.** `--json` is a *global* flag
available on every subcommand (from `bd create --help` Global Flags
on 2026-05-25, identical on `ready`, `list`, `show`, `close`,
`update`). The symphony adapter already exploits this — `beads.go`
line 82 returns `[]string{"--json"}` from `baseArgs()` and prepends it
to every invocation. Pros: ships today, structured output. Cons: ~ms
process start; JSON schema is not itself versioned.

**(b) Programmatic transport.** `bd --help` lists no daemon, gRPC, or
HTTP server. `bd init --server` selects an external Dolt sql-server
backend, not an RPC for `bd` itself. **Not available today.**

**Recommendation: v0.1 ships on (a) shell-out + `--json`-everywhere.**
This is materially better than the task framing assumed — structured
output is here today, not a future upstream ask. The "propose JSON
flag upstream" follow-up disappears. If upstream later exposes a Go
SDK or RPC, swap the transport behind the same `Client` surface.

---

## 4. Module layout

The right mental model is "extract symphony's `internal/tracker/beads/`
package, plus the parts of `internal/tracker/` and `internal/core/` it
depends on, into a standalone module." v0.1 is mostly extraction with
import-path rewrites, not new code.

### Lift map (symphony → beads-go)

| Symphony source                                          | beads-go destination                          |
|----------------------------------------------------------|-----------------------------------------------|
| `internal/tracker/beads/runner.go`                       | `exec.go` (verbatim, drop symphony import)    |
| `internal/tracker/beads/errors.go`                       | `errors.go` (decide envelope shape — see §7)  |
| `internal/tracker/beads/parse.go`                        | `issue.go` + `parse.go`                       |
| `internal/tracker/beads/beads.go`                        | `client.go` + per-verb files                  |
| `internal/tracker/tracker.go` (`TrackerReader/Writer`)   | `writer.go` + `reader.go` (or external — §15) |
| `internal/core/issue.go` (`Issue`, `Priority`, `State`)  | `issue.go`                                    |

### File-by-file proposal

- `client.go` — `Client` and `NewClient(opts Options)`. `Options`
  carries `BinaryPath` (default empty → PATH lookup at exec time),
  `WorkingDir`, `DataDir` (→ `--db=<path>`), `Actor` (→ `--actor=<n>`),
  `ListLimit`, `Logger`. Mirrors symphony's `Config` at
  `internal/tracker/beads/beads.go` lines 21-47.
- `issue.go` — `Issue`, `Priority` enum, `IssueState` typed string,
  `Label`, `Type`, `DepKind` covering
  `blocks|tracks|related|parent-child|discovered-from|until|caused-by
  |validates|relates-to|supersedes` (per `bd dep add --help`
  2026-05-25). Lifted from `internal/core/issue.go`.
- `parse.go` — `beadsIssue` JSON shape and `decodeIssues` /
  `decodeShow`. Lifted from `internal/tracker/beads/parse.go`.
  Field-naming note: bd's JSON uses `"status"`; the converter maps
  that to the public `IssueState` field at the boundary.
- `ready.go`, `list.go`, `show.go` — read operations.
- `create.go` — `Create(ctx, CreateOptions) (id string, err error)`.
  `CreateOptions` carries `Title`, `Description`, `Priority` (the
  client serializes to bd's `"0".."4"` string form per `bd create
  --help`), `Labels`, `Type`, `Silent` (always true when capturing
  ID), plus optionals `Acceptance`, `Notes`, `Parent`, `Deps`,
  `Estimate` (→ `-e <minutes>`), `BodyFile`.
- `close.go` — `Close(ctx, id, reason)`. `--reason=<r>` forwarded
  only when non-empty (symphony lines 324-341).
- `update.go` — `Update(ctx, id, UpdateOptions)` covering `--status`,
  `--add-label`, `--remove-label`, `--set-labels`, `--priority`,
  `--append-notes`, `--assignee`, plus first-class `Claim(ctx, id)`
  wrapping `bd update --claim` (both repos' `CLAUDE.md` mandate it).
- `delete.go` — `Delete(ctx, id, opts DeleteOptions{Force bool})`.
- `dep.go` — `DepAdd(ctx, child, parent, opts ...DepAddOption)`
  defaulting to `DepKindBlocks` (v0.1 hardcodes `blocks`; callers
  needing other kinds pass `WithDepKind(k)`). Plus `DepRemove`,
  `DepTree`, `DepCycles`, `DepList`.
- `graph.go` — `Graph(ctx, opts)` returns raw bytes;
  `GraphOptions.Format` selects `dot|html|box|compact|default`.
- `prime.go` — `Prime(ctx) (string, error)` returns bd's AI-optimized
  markdown context.
- `export.go` — `Export(ctx, opts) (io.Reader, error)` streams JSONL.
- `dolt.go` — `DoltPush(ctx, opts DoltPushOptions{Force bool})` and
  `dolt commit` / `dolt status` shims. Gated behind explicit caller
  invocation; never implicit (§13).
- `writer.go` — `TrackerWriter` interface + `Client` method set.
  Compile-time: `var _ TrackerWriter = (*Client)(nil)`.
- `reader.go` — `TrackerReader` interface + matching methods.
  `FetchCandidates` wraps `bd ready --json`; `FetchByStates` fans out
  per state via `bd list --status=<s> --json` (symphony lines
  148-188); `FetchStatesByIDs` fans out via `bd show <id> --json`.
- `errors.go` — error envelope (§7). Public surface: `*Error`,
  `ErrNotInstalled`, `ErrNotInitialised`, `ErrIssueNotFound`,
  `ErrCycle`, `ErrUnsupported`.
- `exec.go` — `runner` interface + `execRunner`. Lifted verbatim from
  `internal/tracker/beads/runner.go`, including the G204 comment.
- `boundary.go` — `requireID` and `"--"` argv-separator helpers from
  `internal/tracker/beads/beads.go` lines 360-387. Security pattern:
  every positional ID is validated (no empty, no leading `-`) and a
  literal `"--"` separates the flag block from the ID.
- `doc.go` — package overview.

### Optional helper sub-package (deferred to §15)

- `beadstest/` — spins up a real `bd init`-ed sandbox in a tmpdir for
  downstream integration tests. Deferred to v0.2 pending demand.

---

## 5. Public API surface

Sketches — exact signatures land in the v0.1 PR; this is the shape:

```go
opts := beads.Options{
    BinaryPath: "bd",            // empty → PATH lookup at exec time
    WorkingDir: "/path/to/repo",
    DataDir:    "/path/to/repo/.beads/embeddeddolt/foo",
    Actor:      "advisor-session-abc",
    ListLimit:  50,
}
client := beads.NewClient(opts)
```

```go
issues, err := client.Ready(ctx, beads.ReadyOptions{Limit: 10})
id, err := client.Create(ctx, beads.CreateOptions{
    Title:       "Initialize Go module",
    Description: "Set up go.mod, .gitignore, .golangci.yml.",
    Priority:    beads.PriorityCritical,
    Labels:      []string{"impl"},
    Type:        beads.TypeTask,
})
err = client.Close(ctx, id, "Module bootstrapped")
err = client.DepAdd(ctx, child, parent)            // defaults to blocks
err = client.Claim(ctx, id)                        // wraps update --claim
```

`TrackerWriter` / `TrackerReader` parity is a method set on the same
`Client`, satisfying the interface from symphony (or wherever the
interface ends up hosted — §15):

```go
var _ tracker.TrackerWriter = (*beads.Client)(nil)
var _ tracker.TrackerReader = (*beads.Client)(nil)
```

---

## 6. Output parsing strategy

`bd` 1.0.0 emits `--json` on every subcommand inspected
(`ready`, `list`, `show`, `close`, `create`, `update`, `dep *`). The
client parses JSON on every command that produces structured records:

- `ready`, `list`, `show` → `[]beadsIssue` (symphony's `decodeIssues`
  / `decodeShow`). `bd show` emits a single-element array even for one
  ID; the decoder unwraps.
- `create --silent` → stdout is the bare ID, no JSON wrap needed.
  When `--silent` is omitted, `bd create` emits a multi-line summary;
  the client always passes `--silent` when it needs the ID.
- `close` → no structured payload; success is exit code 0.
- `update` → no structured payload by default; `--claim` emits a
  brief acknowledgment that the client ignores (success is exit 0).
- `dep add`, `dep remove` → exit code 0 on success; stderr carries
  cycle-detection messages on failure (mapped per §7).
- `dep tree`, `dep cycles`, `graph` → text/DOT/HTML output; the
  client returns raw bytes for `graph`, and a typed tree for
  `dep tree` (parsed from `--json`).
- `export` → JSONL stream; the client returns an `io.Reader`.
- `prime` → markdown text; returned as a `string`.

Per-subcommand golden files live under `testdata/`. A `bd` minor
release that changes a JSON field name trips the golden file and
fails CI before it ships to consumers. See §12 for the test layout.

---

## 7. Error handling

The client uses a single envelope type `*Error` modeled on symphony's
`tracker.Error` (`internal/tracker/errors.go` lines 180-200). Whether
to lift the entire `Category` taxonomy (`CategoryAPIRequest`,
`CategoryAPIStatus`, `CategoryNotFound`, `CategoryValidation`,
`CategoryTimeout`, `CategoryUnsupported`, ...) or simplify is in §15
as an open question. The recommendation is to lift it, because the
taxonomy was designed cross-adapter and downstream consumers
(symphony's orchestrator) already branch on it.

### Classification rules

- exit code 0, empty stderr → return parsed output, nil error.
- exit code 0, non-empty stderr → return parsed output, log via
  `Options.Logger` at WARN.
- non-zero exit + stderr matches `"not found"|"no such issue"|"issue
  does not exist"` → `*Error{Category: CategoryNotFound,
  Underlying: ErrIssueNotFound}`.
- non-zero exit + stderr matches `"invalid status"|"unknown
  status"|"invalid priority"|"validation failed"` →
  `*Error{Category: CategoryValidation}`.
- non-zero exit + stderr matches `"permission denied"|"unauthorized"
  |"forbidden"` → `*Error{Category: CategoryAuthFailed}` (mostly
  relevant for `bd dolt push` against a remote).
- non-zero exit + stderr matches cycle-detection pattern from `bd
  dep add` (TBD: capture actual stderr in v0.1 golden test, then
  add the pattern) → `*Error{Category: CategoryValidation,
  Underlying: ErrCycle}`.
- non-zero exit otherwise → `*Error{Category: CategoryAPIStatus,
  Status: exitcode}`.
- `*exec.Error` (binary missing / not executable) →
  `*Error{Category: CategoryAPIRequest, Underlying: ErrNotInstalled}`.
- `context.DeadlineExceeded` / `Canceled` →
  `*Error{Category: CategoryTimeout}`.

These are the rules in symphony's `errors.go` lifted directly. The
pattern lists themselves (`notFoundPatterns`, `authPatterns`,
`validationPatterns`) are in `internal/tracker/beads/errors.go`
lines 130-154 and lift verbatim.

### Sentinels

- `ErrNotInstalled` — `bd` not on PATH.
- `ErrNotInitialised` — repo has no `.beads/` directory.
- `ErrIssueNotFound` — issue ID does not exist.
- `ErrCycle` — `bd dep add` would create a cycle.
- `ErrUnsupported` — adapter cannot perform the op (e.g. `LinkPR`).

---

## 8. Concurrency and rate limiting

`bd` is process-based. Concurrent invocations are safe up to the
embedded Dolt store's locking semantics, which serialize writes. The
client offers two modes via `Options.Concurrency`:

- `ConcurrencyUnlimited` (default): no client-side serialization.
  Callers that need ordered writes use their own mutex. This matches
  the symphony adapter today, which is documented as "safe for
  concurrent use within a single process — methods are stateless
  beyond the immutable Config and the injected runner" (`beads.go`
  lines 49-55).
- `ConcurrencyOne`: client-side mutex serializing every `bd`
  invocation. Useful for callers that hit Dolt-locking errors and
  prefer simplicity over throughput.

### Known race: `bd update --claim`

The atomic claim verb is the single most race-sensitive operation
(both consumers' `CLAUDE.md` documents it as "atomically claim"). bd
itself implements atomicity at the database layer; the client just
forwards. The `*Error` envelope carries the bd exit code so callers
can detect a double-claim and back off. v0.1 documents this; later
versions can add a `ClaimResult` typed return that distinguishes
"claimed by me" / "already claimed by me" / "claimed by other."

### Rate limit surface

`RateLimit()` returns the zero `RateLimitSnapshot` (no remote budget).
The interface contract in `~/git/local-symphony/internal/tracker/tracker.go`
line 49 documents this case explicitly.

---

## 9. Migration plan: `local-symphony`

The substantive migration is symphony's, because that is where the
extracted code came from.

a. Confirm symphony's `go.mod` Go version is at least the floor
   `beads-go` targets (proposed: Go 1.22; symphony is already on
   1.25, advisor is on 1.25 — pick the *lower* of the two as the
   floor so future consumers have headroom).
b. Add `github.com/mattsp1290/beads-go` to symphony's `go.mod`. Tag
   `beads-go` `v0.1.0` first (see §11).
c. Replace `internal/tracker/beads/` with a thin shim that
   `import beads "github.com/mattsp1290/beads-go"` and wraps a
   `beads.Client` to satisfy the in-tree `tracker.Tracker` interface.
   Keep the shim because symphony's interface might stay in-tree
   (see §15).
d. If symphony decides to drop its `internal/tracker/` interface and
   import `beads.TrackerWriter` / `beads.TrackerReader` directly,
   the orchestrator and `tracker_write` tool both need import-path
   updates. This is the largest single change in the migration; do
   it in its own PR.
e. Symphony's `test/integration/e2e/beads.go` line 68
   (`exec.LookPath("bd")`) and `test/conformance/tracker/
   helpers_test.go` line 141 (`filepath.Join(dir, "bd")`) keep
   their local exec lookups — they are testing that bd is
   installed in CI, not driving issue operations.
f. Symphony's adapter capability matrix (referenced as
   `docs/tracker-adapters.md` in the package doc) gets updated to
   reflect that the beads adapter now lives at
   `github.com/mattsp1290/beads-go`.
g. Delete symphony's lifted code only after the shim is green for
   a full CI cycle and a manual `bd ready` / `bd close` round-trip.

---

## 10. Migration plan: `advisor`

**Reframed.** The task assumed advisor has in-tree Go shellouts to
`bd` that need migration. It does not — greps for `exec.Command.*"bd"`
across `~/git/advisor/internal` and `~/git/advisor/cmd` come back
empty (verified 2026-05-25). Advisor's only `bd` references are:

- `~/git/advisor/CLAUDE.md` and `AGENTS.md` — agent guidance, not Go.
- `~/git/advisor/setup-beads.sh` — one-off bootstrap shell script;
  stays a shell script.

So this is **enablement, not migration**. Once `beads-go` is tagged,
advisor can adopt it for new programmatic flows: session-completion
automation (file follow-up issues via `beads.Client.Create`, close
the session bead via `beads.Client.Close`) and MCP-tool-failure
issue emission with `discovered-from` deps. Neither is required for
v0.1; ship `beads-go` first and let advisor opt in when a concrete
flow needs it.

---

## 11. Versioning and release plan

`bd` is v1.0.0, so flag stability has a semver promise. `beads-go`
versions itself independently:

- **v0.1.0**: shell-out + `--json`. Verbs both consumers use today —
  `Ready`, `List`, `Show`, `Create`, `Close`, `Update` (with `Claim`
  helper), `Delete`, `DepAdd`, `DepRemove`, `DepCycles`, `Graph` (raw
  bytes). `TrackerReader` / `TrackerWriter` per §15.
- **v0.2.0**: remaining surface — `DepTree` (typed), `Export`
  (streaming), `Prime`, `DepRelate`, `Comment`, `Tag`.
- **v0.3.0**: `bd dolt push` / `dolt commit` shims, metadata helpers,
  optional `beadstest/` sub-package.
- **v0.4.0**: programmatic transport if upstream ships one.
- **v1.0.0**: tag after ≥3 months of consumer stability AND no
  flag-breaking `bd` release in that window.

License: MIT (already present at `/Users/punk1290/git/beads-go/LICENSE`).

---

## 12. Test strategy

Three layers:

### Unit tests with a fake runner

The `runner` interface (lifted from
`internal/tracker/beads/runner.go`) is the seam. Tests construct a
`fakeRunner` returning canned stdout/stderr/exit-code triples; no
`bd` process spawns. Symphony's existing test files
(`internal/tracker/beads/beads_test.go`, `parse_test.go`,
`errors_test.go`) lift over with minor import-path changes and
cover most of the verb surface already.

### Integration tests against a real `bd` binary

A `//go:build integration` suite runs `bd init` in a tmpdir, then
exercises the full Create → Show → Update → Close → DepAdd loop.
CI installs `bd` via a setup step modeled on
`~/git/advisor/setup-beads.sh` and `~/git/local-symphony/
setup-beads.sh`. The install step pins a `bd` minimum version
(initially 1.0.0) via a `BD_MIN_VERSION` env var the test reads at
startup and skips/fails if mismatched.

### Golden files per subcommand

`testdata/golden/<subcommand>.json` captures the structured payload
each subcommand emits, recorded from a real `bd` run. The unit
tests' fake runner returns these payloads byte-for-byte; the
integration suite re-runs the same subcommand against a fresh
sandbox and diffs the output against the golden file. A bd minor
release that changes a JSON field name trips the golden test and
fails CI before downstream consumers see the change.

### Property test

A `TestRoundTripCreateShowClose` property test creates a random
issue (random Priority / Labels / Type / Description), shows it,
asserts the round-trip preserves fields, closes it with a random
reason, shows it again, asserts state == closed and reason
preserved.

---

## 13. Risk register

| Risk                                | Severity | Mitigation                                                            |
|-------------------------------------|----------|-----------------------------------------------------------------------|
| `bd` flag rename in 1.x             | Medium   | Golden files + min-version pin (`BD_MIN_VERSION`) + semver tracking   |
| `bd` JSON field rename in 1.x       | Medium   | Golden files catch this in CI before consumers see it                 |
| Process-start cost in tight loops   | Low      | Document. Recommend `Ready` / `List` over fan-out `Show`              |
| Concurrent `bd update --claim`      | Medium   | Document. `ConcurrencyOne` mode for sensitive callers. §8             |
| `bd` not on PATH                    | Low      | `ErrNotInstalled` sentinel; PATH lookup at exec time, not New() time  |
| `.beads/` missing                   | Low      | `ErrNotInitialised` sentinel; document `bd init` as prerequisite      |
| `bd dolt push` side effects         | High     | Gated behind explicit `DoltPush` call. Never invoked implicitly       |
| `bd` rename / fork                  | Low      | Module path `github.com/mattsp1290/beads-go` is stable                |
| Argv injection via issue IDs        | Medium   | `requireID` rejection + `"--"` separator (lifted from §4 `boundary.go`)|
| Dolt-store lock contention          | Low      | Retry policy via `*Error.IsRetryable()`; default `ConcurrencyUnlimited` |
| Schema drift on optional fields     | Low      | `beadsIssue` ignores unknown JSON fields by design                    |

The task's "Beads is pre-1.0 ... contract will churn" framing
overstates the risk: `bd 1.0.0` is the installed reality, so flag
breakage carries a semver penalty upstream. JSON additions remain
unsafe without golden coverage.

---

## 14. First-PR breakdown

Five PRs, in order:

- **PR1: Skeleton.** `go.mod` (module
  `github.com/mattsp1290/beads-go`, Go 1.22), `LICENSE` (already
  present), `README.md` (already present, expand), GitHub Actions
  matrix with a `bd` install step that mirrors
  `~/git/local-symphony/setup-beads.sh`'s install logic.
  `doc.go`, `Makefile` targets for `test`, `test-integration`,
  `lint`.
- **PR2: Core client + read verbs.** `client.go`, `exec.go`,
  `errors.go`, `boundary.go`, `issue.go`, `parse.go`, `ready.go`,
  `list.go`, `show.go`. Lifted from symphony with import-path
  rewrites. Unit tests and golden files for the three read verbs.
- **PR3: Write verbs.** `create.go`, `close.go`, `update.go`
  (including `Claim`), `delete.go`, `dep.go` (`DepAdd`,
  `DepRemove`, `DepCycles`). Unit tests + integration test
  covering the Create → Claim → Close round-trip.
- **PR4: TrackerReader/Writer + tag v0.1.0.** `reader.go`,
  `writer.go`, compile-time interface assertions. CI runs the
  full integration suite against a real `bd 1.0.0`. Tag v0.1.0.
- **PR5: Symphony migration.** In `~/git/local-symphony`,
  replace `internal/tracker/beads/` with a thin shim wrapping
  `beads.Client`. Keep symphony's `internal/tracker/` interface
  (or move it to `beads-go` per §15 — separate PR).

PR5 is in symphony, not in `beads-go`. Advisor adoption is
deferred to a later PR contingent on a concrete user-facing flow
needing it (§10).

---

## 15. Open questions / TODOs for human review

- **License.** MIT already in `LICENSE`. Confirm.
- **Programmatic transport.** Not in `bd 1.0.0`. Recheck per minor release.
- **Interface hosting.** `TrackerWriter` / `TrackerReader` in
  `beads-go` vs. a separate `eino-tools/tracker` module. In
  `beads-go`: simplest layering. In `eino-tools/tracker`: lets future
  Linear / JIRA adapters live there without importing `beads-go`.
  Recommendation: ship them *in* `beads-go` for v0.1; if a third
  adapter materializes, extract.
- **`beadstest/` sub-package.** Real `bd init` sandbox for downstream
  integration tests. Defer to v0.3 pending demand.
- **Multiple projects per `Client`.** v0.1 models one `Client` = one
  `DataDir`. Revisit if a consumer asks.
- **Raw `bd graph` output.** v0.1 returns raw bytes. A typed `Graph`
  struct could land later if a consumer needs it.
- **`Category` taxonomy.** Lift symphony's full enum (recommended) or
  start smaller. The taxonomy is cross-adapter by design.
- **`ClaimResult` typed return.** v0.2 — distinguish "claimed by me"
  / "already claimed by me" / "claimed by other".
- **Module-name conflict.** If upstream ships a Go SDK as
  `github.com/gastownhall/beads`, `beads-go` coexists at its own
  module path; package name stays `beads`.

---

Plan lives at `/Users/punk1290/git/beads-go/docs/prompts/extract-beads-go.md`;
next is PR1 (skeleton + LICENSE + go.mod + GitHub Actions matrix with
a `bd 1.0.0` install step).
