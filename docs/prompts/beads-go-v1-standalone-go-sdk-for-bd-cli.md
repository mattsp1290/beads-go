# Project Planning with Beads

## Agent Instructions

You are an expert software architect creating a comprehensive task breakdown. This task graph will be executed by AI agents working in parallel, coordinated through MCP Agent Mail with file reservations to prevent conflicts.

<quality_expectations>
Create a thorough, production-ready task graph. Include all necessary setup, implementation, testing, and documentation tasks. Go beyond the basics - consider edge cases, error handling, security considerations, and integration points. Each task should be specific enough for an agent to execute independently without ambiguity.
</quality_expectations>

## Project Information

### Links to Relevant Documentation

- **Shared-modules research set** — `~/docs/eino/index.html` (executive summary of the four-module family) and `~/docs/eino/05-shared-repos-proposal.html` (the section flagging `beads-go` as a "next-up candidate"). `~/docs/eino/02-shared-eino-patterns.html` and `~/docs/eino/04-integration-plan.html` give wider context on why the family exists.
- **Existing implementation to lift from** — `~/git/local-symphony/internal/tracker/beads/{beads.go,runner.go,parse.go,errors.go,doc.go}` plus its tests. Production-grade and already test-seamed; the v1 SDK is essentially a generalize-and-extract of this code.
- **Interface contract to deliberately NOT inherit** — `~/git/local-symphony/internal/tracker/tracker.go` defines the `TrackerReader / TrackerWriter / Tracker` triad. Read it to understand what local-symphony needs from the SDK, but do **not** re-export the triad — it's local-symphony's tracker-abstraction concern, not the SDK's.
- **Downstream consumer of the writer surface** — `~/git/local-symphony/internal/worker/tools/trackerwrite/trackerwrite.go`. Confirms which writer ops are actually exercised in production (only `Close`; `Comment`, `Transition`, `LinkPR` are zero-caller surfaces).
- **`bd` CLI conventions** — `~/git/dotfiles/.agents/rules/beads.md` documents the full command set, the priority numbering (0=critical..4=backlog), labels, and the `bd dep add CHILD PARENT` convention. Useful as the canonical reference for command-shape decisions.
- **Upstream `bd` project** — `gastownhall/beads` (referenced in `~/git/local-symphony/README.md`). Source of truth for JSON-output shapes and any future JSON-RPC surface.
- **Verified non-consumer** — `~/git/advisor`. Grep confirms zero programmatic `bd` use today (only `CLAUDE.md` operator instructions and doc-comment bead IDs). v1 has exactly one programmatic consumer: `local-symphony`.

### Project Description

**`beads-go`** is a standalone, reusable Go SDK for the `bd` (beads) task-graph CLI. It is the fifth member of the `mattsp1290` shared-modules family (alongside `codex-auth-go`, `eino-providers`, `eino-tools`, `agent-otel`) — published at `github.com/mattsp1290/beads-go` so future agentic projects can drop it in instead of each rolling their own bd adapter.

**Scope reframe (verified against code, not the proposal):** The original eino research doc claims two consumers (advisor + local-symphony); grep against `~/git/advisor` shows zero programmatic `bd` use. v1 is sized to one real consumer — `local-symphony` — with the explicit, testable acceptance criterion that `local-symphony/internal/tracker/beads/` can be replaced with `import "github.com/mattsp1290/beads-go/beads"` plus a thin wrapper satisfying `local-symphony`'s internal `tracker.Tracker` interface.

**v1 public surface (deliberately small):**

- Single package `github.com/mattsp1290/beads-go/beads`. No premature split into sub-packages.
- Concrete `Client` struct (not an interface). Constructed via functional options:
  ```go
  client, err := beads.NewClient(
      beads.WithBinary("/usr/local/bin/bd"),
      beads.WithDataDir("/var/lib/symphony/beads/proj"),
      beads.WithActor("orchestrator"),
      beads.WithListLimit(50),
  )
  ```
- **Six methods**: `Ready(ctx, opts...) ([]Issue, error)`, `List(ctx, opts...) ([]Issue, error)` (state, label, and `WithAll()` filters as options, not positionals, so adding more is non-breaking), `Show(ctx, id) (Issue, error)` (returns `ErrNotFound` on miss — matches `os.Open` shape), `Close(ctx, id, reason string) error`, `Comment(ctx, id, body string) error`, and `Transition(ctx, id, state string) error`.
- **Deliberately deferred** from v1, each with a concrete promotion trigger:
  - `LinkPR` — bd has no PR concept; local-symphony's wrapper returns its existing unsupported error for this tracker-specific interface method.
  - `Create`, `Update`, `Dep add/remove/tree`, `Graph`, `Init`, `Remember`, `Prime`, `Dolt push` — operator/developer workflow today; promote when the first Go program programmatically authors graphs or bootstraps repos.
  - `RateLimit() RateLimitSnapshot` — vestigial for a local CLI (the existing adapter returns the zero value). Future RPC rate limits should surface inline as a call error when the operation hits one, not via a separate poll method.
- **Hidden transport + runner seams**: the SDK has an unexported typed `transport` interface owned by `Client`, with `execTransport` as the only built-in for v1. `execTransport` has a smaller unexported runner seam for `os/exec` so unit tests can assert exact argv. Do not export either seam; do not freeze a public operation enum or wire envelope.
- **Slim 8-kind `Error` envelope** with sentinel support:
  ```go
  type Error struct {
      Op     string  // "Ready", "Close", ...
      Kind   Kind    // KindNotFound, KindValidation, KindTimeout,
                    //   KindTransport, KindAuthFailed, KindUnsupported,
                    //   KindBadResponse, KindExit
      Status int     // exec exit code or future HTTP status; 0 if N/A
      Err    error   // wrapped underlying
  }
  var ErrNotFound, ErrUnsupported, ErrValidation = errors.New(...)
  ```
  10-category taxonomy from local-symphony's `tracker.Error` is not re-exported. SDK classification is still precise enough for the wrapper to map `KindBadResponse` to `CategoryUnknownPayload`, `KindAuthFailed` to `CategoryAuthFailed`, `KindExit` to `CategoryAPIStatus`, and transport/timeouts/validation/not-found/unsupported to their local equivalents. Define exact mappings in tasks:
  - `context.DeadlineExceeded` and `context.Canceled` -> `KindTimeout`
  - binary start/look-path/path errors -> `KindTransport`
  - stderr containing observed auth patterns (`permission denied`, `unauthorized`, `forbidden`) -> `KindAuthFailed`
  - stderr containing observed not-found patterns -> `KindNotFound`
  - boundary validation or observed validation stderr -> `KindValidation`
  - malformed JSON, unsupported JSON envelope, missing required `id`/`status`, wrong `show` ID -> `KindBadResponse`
  - unclassified non-zero exit -> `KindExit` with `Status`
- **`Issue` type** is normalized but opinion-light:
  - Typed `time.Time` (UTC) for timestamps; SDK does the RFC3339 parse so callers don't repeat it. Missing or malformed timestamps decode to the zero time and do not fail the issue.
  - `Priority *int` carrying bd's native 0-4 while preserving omitted/unset priority. No named enum — that's a consumer-domain opinion.
  - Labels passed through as bd sends them (no lowercase normalization — that's local-symphony's cross-tracker-comparison choice).
  - `Dependencies []Dependency`, preserving both bd JSON shapes: `{issue_id, depends_on_id, type}` from ready/list and `{id, dependency_type}` from show. The SDK preserves raw dependency kind and IDs; local-symphony maps only kind `blocks` to `core.Issue.BlockedBy`, drops self references, dedupes, and sorts.
  - `RawJSON json.RawMessage` escape hatch carrying the exact JSON object for this issue (not the enclosing list/envelope) so callers can decode bd-added fields before the SDK ships a typed accessor.
  - Package-level forward-compat contract: *unknown JSON fields are silently preserved in `RawJSON` and dropped from typed fields; adding a typed field is non-breaking; renaming or retyping one is breaking.*
  - Decoder handles both current legacy JSON arrays/objects and the upstream `BD_JSON_ENVELOPE=1` shape (`{"schema_version":N,"data":...}`), with unit coverage for both. Unsupported envelope versions are `KindBadResponse`.
- **Context-only timeouts.** Every method takes `ctx` first. No `WithDefaultTimeout` option in v1 (adding later is non-breaking; surprising callers who already set a deadline is not).
- **Single package-level concurrency contract:** "A `*Client` is safe for concurrent use by multiple goroutines because its config is immutable after construction. The SDK does not serialize calls per data directory. Multi-process and multi-writer safety against the same `.beads/` data directory depends on bd's embedded-Dolt file lock; callers running other `bd` writers concurrently are responsible for workflow-level serialization." Lift the single-writer discussion from local-symphony's package doc into the SDK docs.

**Non-negotiable security contract (lifted verbatim from the local-symphony adapter):**

- `requireID` boundary check rejecting empty IDs and any ID starting with `-` (CategoryValidation in the existing adapter; `KindValidation` here) for every public method that accepts a positional ID.
- Literal `--` argv separator between flags and every positional ID on every exec call.
- `#nosec G204` annotation + comment block on the exec runner explaining why exec-with-non-constant-args is intentional (operator-configured binary + internal-only argv construction + no shell).
- Both layers stay together — they're defense-in-depth, not redundancy.

**Family conventions** (same as the four eino-spine modules):

- MIT license (LICENSE file in repo root).
- `main` default branch (no `master`).
- `docs/adr/` directory carrying design decisions worth preserving (Issue-type normalization, hidden transport interface, v1 method scope, exec security, error taxonomy, local-symphony adapter contract, etc.).
- Hand-curated `CHANGELOG.md` on the way to v1.0.0; switch to `release-please` once tagged.
- README opens with one sentence on what it does, one example, link to consumer projects. No marketing.
- `.golangci.yml` baseline matching `~/git/local-symphony/.golangci.yml`.
- Pre-v1.0 semver caveat: breaking changes allowed under v0.x (this is explicitly a `v0` library while the JSON-RPC transport question is still open).

### Technical Stack

- **Language:** Go 1.26. The module's `go` directive pins 1.26. No multi-version support claim.
- **Dependencies:** stdlib only in v1 — `os/exec`, `encoding/json`, `context`, `errors`, `time`, `strings`, `sort`, `fmt`. No third-party Go deps (no cobra, no zap, no gjson, no flock — the package is a pure library, not a CLI or a daemon).
- **Module path:** `github.com/mattsp1290/beads-go`. Single public package at `github.com/mattsp1290/beads-go/beads`.
- **Build/test tooling:** stdlib `go test`, `go vet`, `golangci-lint` (v2.x, baseline from `~/git/local-symphony/.golangci.yml`).
- **License:** MIT.
- **CI:** GitHub Actions, Go 1.26 only (`actions/setup-go` with `1.26.x`, or `go-version-file` if a toolchain file is added). Pipeline: `go mod tidy --diff`, `go vet ./...`, `golangci-lint run`, formatting verification (`gofmt`/`goimports` or `golangci-lint fmt --diff`), `go test -race ./...`, then the bd-binary integration job (see Specific Requirements). No release-automation in v1.
- **Repo layout** (pre-task-graph):
  ```
  github.com/mattsp1290/beads-go/
  ├── LICENSE                 (MIT — verify current text and copyright)
  ├── README.md               (one-sentence purpose + one example + consumer link)
  ├── CHANGELOG.md            (hand-curated, starts at Unreleased)
  ├── go.mod                  (go 1.26, no deps)
  ├── .golangci.yml           (lifted from local-symphony, paths fixed up)
  ├── .github/workflows/      (single ci.yml — see Specific Requirements)
  ├── beads/                  (the public package)
  │   ├── doc.go              (package-level concurrency + forward-compat contract)
  │   ├── client.go           (Client struct, NewClient, functional options)
  │   ├── options.go          (Client options + list options: WithState/WithLabel/WithAll)
  │   ├── issue.go            (Issue type, wire decode, RawJSON)
  │   ├── errors.go           (Error envelope, Kind, sentinels, classification)
  │   ├── transport.go        (unexported typed transport interface + execTransport)
  │   ├── runner.go           (unexported exec runner seam; #nosec G204 lives here)
  │   ├── ready.go            (Client.Ready)
  │   ├── list.go             (Client.List + list-options)
  │   ├── show.go             (Client.Show)
  │   ├── close.go            (Client.Close)
  │   ├── comment.go          (Client.Comment)
  │   ├── transition.go       (Client.Transition)
  │   └── *_test.go           (fake-transport unit tests for every method)
  ├── internal/
  │   └── argvsec/            (requireID + "--" separator helpers; #nosec G204 lives here)
  └── docs/
      └── adr/                (0001-flat-client.md, 0002-hidden-transport.md, ...)
  ```

### Specific Requirements

1. **CI installs the real `bd` binary via `npm install` and runs integration tests against it.** A dedicated GitHub Actions job (separate from the unit-test job) sets up Node, runs `npm install -g <beads-package-name>` (resolve the actual npm package name from `gastownhall/beads/npm-package/package.json`; expected name `@beads/bd`, bin `bd`; decide whether CI pins a known version such as `@beads/bd@1.0.4` or intentionally floats latest, and document the choice in the task), verifies `bd --version` is on `PATH`, then runs `go test -tags=integration ./...`. Integration tests must:
   - Create a temp dir, run `git init`, then run `bd init --non-interactive --skip-agents --skip-hooks --prefix bgotest --database bgotest`. Run SDK calls with `WithDataDir` pointing at the temp repo's `.beads` path or by setting the command working directory to the temp repo.
   - Exercise the full v1 method set (`Ready`, `List`, `Show`, `Close`, `Comment`, `Transition`) end-to-end through the real binary.
   - Cover real-binary success paths and stderr classes that bd can naturally emit, especially `KindNotFound` from `bd show <bogus-id>` and at least one stable CLI validation failure if available.
   - Unit tests, not real-binary integration tests, cover boundary validation without runner invocation, timeout via fake/hanging runner, `KindBadResponse` via malformed JSON/wrong shape/wrong returned ID fixtures, `KindUnsupported`, and `KindTransport` via missing/non-executable binary or fake runner.
   - Cover both default JSON output and the `BD_JSON_ENVELOPE=1` envelope shape, either through integration matrix entries or targeted unit fixtures.
   - Run cleanly in a fresh container — no assumptions about pre-existing bd state.
2. **Security contract verbatim:** `requireID`'s `-`-prefix guard and the `--` argv separator on every positional ID. The `#nosec G204` annotation + comment block stays on the exec runner. A unit test asserts that passing `id := "-rf"` to every public method that accepts a positional ID (`Show`, `Close`, `Comment`, `Transition`, and any future ID-taking methods) returns `KindValidation` without invoking the runner (the test uses a runner fake that panics if called).
3. **Acceptance criterion = the local-symphony replacement PR.** v1 is not done until a PR against `~/git/local-symphony` replaces the existing bd exec adapter with a local wrapper around `*beads.Client` and all updated local-symphony tests pass. Prefer preserving the existing package path `internal/tracker/beads` if that minimizes churn in runtime wiring and conformance tests; otherwise include the runtime import updates. Before the module is tagged, validate with a temporary `replace github.com/mattsp1290/beads-go => ../beads-go` in local-symphony, and remove the replace after the `v0.1.0` tag is available. This is the v1 success gate.

   The wrapper must preserve these local-symphony-specific behaviors:
   - `FetchCandidates`: call SDK `Ready`, filter by `activeStates` client-side, return a non-nil empty slice, and still invoke `Ready` when `activeStates` is empty.
   - `FetchByStates`: empty/nil/all-empty states short-circuit without invoking bd; otherwise fan out one SDK `List` call per state using `WithState(state)` and `WithAll()`, then dedupe by ID with first-state-wins.
   - `FetchStatesByIDs`: empty input returns an empty map without invoking bd; dedupe IDs; skip empty IDs; tolerate `ErrNotFound` by omitting that ID; return a partial map on hard errors; reject mismatched echoed IDs as local `CategoryUnknownPayload`.
   - Issue mapping: priority `nil`/`0..4` maps to `core.Priority` including unset/backlog; labels lowercase/trim/dedupe/sort; dependencies kind `blocks` map to `BlockedBy` across both bd JSON shapes, dropping self refs; timestamps are UTC and malformed timestamps remain zero.
   - Writers: `Close` trims reason and omits `--reason` for empty reason; `Comment` and `Transition` preserve the current local-symphony behavior by composing SDK methods; `LinkPR` remains unsupported in the wrapper.
   - `RateLimit` returns the zero `tracker.RateLimitSnapshot`.
   - SDK errors are mapped into `tracker.Error` categories, including `KindBadResponse` -> `CategoryUnknownPayload` and `KindAuthFailed` -> `CategoryAuthFailed`.
4. **Forward-compat contract documented at package level:** unknown JSON fields preserved in per-issue `RawJSON`, typed-field additions non-breaking, typed-field renames/retypes breaking. Unit tests assert that decoding bd payloads with unknown top-level fields inside issue objects succeeds and that the unknown fields survive in each issue's `RawJSON` for `Ready`, `List`, and `Show`.
5. **No `Raw(args ...string) ([]byte, error)` escape hatch on `Client`.** Tempting for v1 to cover gaps; shipping it publishes a CLI-shaped contract that any future JSON-RPC transport will have to either reject or emulate. Don't ship it.
6. **Replay-style runner fake for unit tests.** Tests use a fake exec runner beneath `execTransport` that returns canned stdout/stderr/exit-code per scripted invocation, asserting on the exact argv constructed (including the `--json`, `--db=`, `--actor=`, `--`, `-n`, `--all`, `--reason=`, `--append-notes=`, and `--status=` placement). No test depends on the real `bd` binary except the integration job in #1.
7. **Pre-v1.0 versioning:** the module ships under `v0.x` tags. Breaking changes are allowed between minor versions until JSON-RPC transport question is resolved and a v1.0.0 is cut. `CHANGELOG.md` documents every breaking change with a migration note.
8. **No telemetry, no logging.** The library does not import `log`, `slog`, or any OTel package. Callers wrap. (Consistent with `agent-otel` being a separate spine module.)
9. **`golangci-lint` baseline matches `~/git/local-symphony/.golangci.yml`, retargeted for this module.** The shared modules feel like one family; the linter config is part of that. Explicitly set `run.go: "1.26"`, set `goimports.local-prefixes: github.com/mattsp1290/beads-go`, remove local-symphony-specific path exclusions such as `internal/runtime`, and add a formatting verification step.
10. **Docs and release-readiness are first-class tasks.** Include ADR tasks for `0001-v1-scope.md`, `0002-json-shape-and-rawjson.md`, `0003-exec-security.md`, `0004-error-taxonomy.md`, and `0005-local-symphony-adapter.md`; README notes the supported/tested bd package/version and install command; `CHANGELOG.md` starts with `Unreleased`; release checklist covers `v0.1.0` tag and pkg.go.dev verification.

---

## Your Task

Analyze this project and create a comprehensive **Beads task graph** using the `bd` CLI. Beads provides dependency-aware, conflict-free task management for multi-agent execution.

---

<critical_constraint>
Your ONLY output is a bash shell script. Do NOT use `bd add` — the correct command to create a bead is `bd create`. Use `bd dep add` for dependencies. Do not implement anything yourself.
</critical_constraint>

## Output Format

Generate a shell script that creates the full task graph. The script should:

1. **Initialize Beads** (if not already initialized)
2. **Create all beads** with appropriate priorities
3. **Establish dependencies** between beads
4. **Add labels** for phase grouping

### Example Output

```bash
#!/usr/bin/env bash
# Project: beads-go
# Generated: 2026-05-25

set -euo pipefail

# Initialize beads if needed
if [ ! -d ".beads" ]; then
    bd init --non-interactive
fi

echo "Creating project beads..."

# ========================================
# Phase 1: Project Setup & Infrastructure
# ========================================

SETUP_GOMOD=$(bd create "Initialize Go module at github.com/mattsp1290/beads-go with go 1.26" \
  -d "Run 'go mod init github.com/mattsp1290/beads-go'. Set go directive to 1.26. No third-party deps in v1; go.sum should remain empty." \
  -p 0 -l setup --silent)

SETUP_LICENSE=$(bd create "Verify LICENSE contains MIT text for 2026 Matt Spurlin" \
  -d "Verify the existing LICENSE uses the standard MIT template. Year 2026, copyright holder 'Matt Spurlin'. Update only if the current file is a placeholder or stale." \
  -p 1 -l setup --silent)

SETUP_LINT=$(bd create "Add .golangci.yml mirroring local-symphony's v2.x baseline" \
  -d "Copy ~/git/local-symphony/.golangci.yml, retarget run.go to 1.26, set goimports.local-prefixes to github.com/mattsp1290/beads-go, remove local-symphony-specific path skips, and add formatting verification." \
  -p 1 -l setup --silent)
bd dep add "$SETUP_LINT" "$SETUP_GOMOD"

# ... continue for all phases ...

echo ""
echo "Bead graph created! View with:"
echo "  bd ready              # List unblocked tasks"
```

---

## Bead Creation Guidelines

### Priority Levels
- `-p 0` = Critical (blocking other work)
- `-p 1` = High (important but not blocking)
- `-p 2` = Medium (standard work)
- `-p 3` = Low (nice to have)
- `-p 4` = Backlog / lowest priority

### Labels (Phase Grouping)
Use `-l` or `--labels` to group beads by phase. Do not use `--label`.
- `setup` - Project initialization
- `core` - Core architecture
- `feature-{name}` - Feature-specific work
- `testing` - Test coverage
- `docs` - Documentation
- `deploy` - Deployment/CI

### Dependency Rules
1. Never create cycles
2. Every bead should have a clear dependency chain back to setup tasks
3. Use `bd dep add CHILD PARENT` (child depends on parent completing first)
4. Quote captured bead IDs in dependency commands: `bd dep add "$CHILD" "$PARENT"`
5. Parallel work should share a common ancestor, not depend on each other

### Task Granularity
- Each bead should be completable in **under 750 lines of code**
- Tasks should be atomic enough for one agent to complete without coordination
- If a task requires multiple file areas, consider splitting by file area

---

## File Reservation Planning

For each major work area, note the file patterns that will need exclusive reservation:

```bash
# Example reservation notes (add as bead descriptions)
# Core public surface: beads/client.go, beads/options.go, beads/doc.go
# Wire decode: beads/issue.go, beads/*_test.go (issue-related)
# Transport/runner: beads/transport.go, beads/runner.go, internal/argvsec/**
# Errors: beads/errors.go, beads/errors_test.go
# Per-method: beads/{ready,list,show,close,comment,transition}.go + matching tests
# CI / lint config: .github/workflows/**, .golangci.yml
# Integration tests: integration/** (build tag: integration)
# Consumer-side acceptance PR: lives in ~/git/local-symphony, not this repo
```

This helps agents claim appropriate file surfaces when they start work.

---

## Context Documentation

Place any important context in `prompts/docs/` for agents to reference. This includes:
- Architecture decisions
- API documentation
- Design system specs
- External service integration guides

---

## Verification Steps

After generating the script:

1. **Run it**: `chmod +x setup-beads.sh && ./setup-beads.sh`
2. **Check ready work**: `bd ready` should show initial setup tasks

---

## Completeness Checklist

Ensure your task graph includes:

- [ ] All setup and configuration tasks
- [ ] Core architecture and shared utilities
- [ ] Feature implementation tasks (broken into small units)
- [ ] Error handling and edge cases
- [ ] Unit and integration tests for each feature
- [ ] API documentation
- [ ] Security considerations (input validation, auth checks)
- [ ] Performance considerations where relevant
- [ ] CI/CD and deployment tasks
- [ ] Clear dependency chains with no cycles
