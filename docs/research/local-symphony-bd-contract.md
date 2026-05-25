# local-symphony bd adapter contract

Date: 2026-05-25

This note records the command and JSON contract that `beads-go` must preserve
for the first consumer, `~/git/local-symphony`. Sources inspected:

- `~/git/local-symphony/internal/tracker/beads/{beads.go,runner.go,parse.go,errors.go,doc.go}`
- `~/git/local-symphony/internal/tracker/beads/*_test.go`
- `~/git/local-symphony/internal/tracker/tracker.go`
- `~/git/local-symphony/internal/worker/tools/trackerwrite/{trackerwrite.go,doc.go,*_test.go}`
- Beads docs at `https://gastownhall.github.io/beads/` and CLI reference pages
  for `list`, `ready`, `show`, `update`, and `close`
- Live local `bd version 1.0.0 (72170267: HEAD@72170267e00a)`

The requested `~/git/dotfiles/.agents/rules/beads.md` path does not exist in
this workspace. A search under `~/git` and `~/.agents` found no replacement
`beads.md` agent rule file.

## Consumer surface

`local-symphony` needs a wrapper around `*beads.Client` that satisfies:

```go
type TrackerReader interface {
	FetchCandidates(context.Context, []core.IssueState) ([]core.Issue, error)
	FetchByStates(context.Context, []core.IssueState) ([]core.Issue, error)
	FetchStatesByIDs(context.Context, []string) (map[string]core.IssueState, error)
	RateLimit() tracker.RateLimitSnapshot
}

type TrackerWriter interface {
	Comment(context.Context, id, body string) error
	Transition(context.Context, id string, toState core.IssueState) error
	Close(context.Context, id, reason string) error
	LinkPR(context.Context, id, prURL string) error
}
```

The `tracker_write` tool exposes one model-facing tool named `tracker_write`
with `op` values `comment`, `transition`, `close`, and `link_pr`. In
local-symphony v1 only `op=close` executes. The other three ops validate
structurally but return `error.category=unsupported_op` before reaching the
writer. `close` forwards `id` and optional `reason` exactly to the writer.

## Command shapes

The SDK should build argv directly with no shell. Global flags come before the
subcommand:

```text
bd --json [--db=<data-dir>] [--actor=<actor>] <subcommand> ...
```

`Config` behavior to preserve:

- Empty binary means run `bd` from `PATH`.
- Non-empty data directory becomes `--db=<path>` on every call.
- Non-empty actor becomes `--actor=<name>` on every call.
- Positive list limit becomes `-n <limit>` on `ready` and `list` calls.
- Deadlines are caller-owned in `beads-go`; local-symphony can wrap calls with
  contexts or its own adapter timeout policy.

Reader commands:

```text
bd --json [--db=<dir>] [--actor=<actor>] ready [-n <limit>]
bd --json [--db=<dir>] [--actor=<actor>] list --status=<state> --all [-n <limit>]
bd --json [--db=<dir>] [--actor=<actor>] show -- <id>
```

Writer commands:

```text
bd --json [--db=<dir>] [--actor=<actor>] close [--reason=<trimmed-reason>] -- <id>
bd --json [--db=<dir>] [--actor=<actor>] update --append-notes=<trimmed-body> -- <id>
bd --json [--db=<dir>] [--actor=<actor>] update --status=<state> -- <id>
```

Security details the SDK must keep:

- Validate write/read IDs before argv construction: trim-space empty IDs are
  validation errors, and IDs starting with `-` are rejected.
- Put a literal `--` before positional IDs so the bd flag parser cannot treat
  an ID as an option.
- Pass `--reason=...`, `--append-notes=...`, and `--status=...` as one argv
  element each.
- Never expose a public raw argv escape hatch.

`LinkPR` is unsupported for beads because bd has no pull-request field.

## Reader semantics

`FetchCandidates`:

- Always calls `Ready`, even when `activeStates` is empty.
- Filters the ready set client-side to issues whose status is in
  `activeStates`.
- Returns a non-nil empty slice when no issue matches.
- Validates that each issue has non-empty `id` and `status`.

`FetchByStates`:

- Empty/nil state input returns a non-nil empty slice without invoking bd.
- Empty state strings are skipped.
- For each non-empty state, call `List` with `--status=<state>` and `--all`.
- Deduplicate by issue ID, first state wins.
- Validate non-empty `id` and `status`.

`FetchStatesByIDs`:

- Empty/nil ID input returns an empty map without invoking bd.
- Empty IDs are skipped and duplicate IDs are called once.
- Fan out with `Show` once per unique ID.
- Not-found responses are tolerated by omitting that ID from the map.
- Hard errors return the partial map plus the error.
- If `bd show <requested>` returns a different non-empty `id`, classify that
  as bad payload rather than mis-attributing state.
- Validate non-empty `id` and `status` before writing the map.

`RateLimit` returns the zero `tracker.RateLimitSnapshot`; `IsUnknown()` should
be true.

## Writer semantics

`Comment`:

- Reject empty/whitespace ID and empty/whitespace body with validation errors.
- Trim the body before sending.
- Append via notes, not `bd comment`, because notes are non-interactive and
  durable.
- Non-idempotent: retry dedupe belongs above the SDK.

`Transition`:

- Reject empty/whitespace ID and empty target state with validation errors.
- Let bd validate unknown status names.

`Close`:

- Reject empty/whitespace ID with validation error.
- Trim reason before sending.
- Omit `--reason` when the trimmed reason is empty.
- Treat bd close as idempotent for local-symphony retry behavior.

## JSON fixtures

`bd ready --json` and `bd list --json` emit arrays. The fields consumed by
local-symphony are:

```json
[
  {
    "id": "beadsgo-nc9",
    "title": "Initialize go.mod for github.com/mattsp1290/beads-go using Go 1.26",
    "description": "Create or verify go.mod with module github.com/mattsp1290/beads-go and go directive 1.26.",
    "status": "open",
    "priority": 0,
    "issue_type": "task",
    "labels": ["setup"],
    "created_at": "2026-05-25T20:45:13Z",
    "updated_at": "2026-05-25T20:45:13Z",
    "dependencies": [
      {
        "issue_id": "beadsgo-nc9",
        "depends_on_id": "beadsgo-85c",
        "type": "blocks"
      }
    ]
  }
]
```

`bd show <id> --json` emits a single-element array. Its dependency shape embeds
issue summaries and uses `dependency_type`:

```json
[
  {
    "id": "beadsgo-nc9",
    "title": "Initialize go.mod for github.com/mattsp1290/beads-go using Go 1.26",
    "status": "open",
    "priority": 0,
    "issue_type": "task",
    "labels": ["setup"],
    "dependencies": [
      {
        "id": "beadsgo-85c",
        "title": "Establish repository baseline for beads-go v0.1.0",
        "status": "closed",
        "priority": 0,
        "issue_type": "task",
        "dependency_type": "blocks"
      }
    ],
    "dependents": [
      {
        "id": "beadsgo-6le",
        "title": "Create public beads package and docs directory layout",
        "status": "open",
        "dependency_type": "blocks"
      }
    ]
  }
]
```

Parser requirements:

- Accept empty, whitespace-only, `null`, and `[]` payloads as empty results.
- Treat malformed JSON or an object where an array is expected as bad payload.
- Ignore unknown fields.
- Required fields for SDK issue records are `id` and `status`.
- `identifier` falls back to `id` when omitted.
- `priority` is `*int`: omitted maps to unset, 0..4 map to the five priority
  levels, out-of-range maps to unset.
- Labels should remain available raw in the SDK. The local-symphony wrapper
  lowercases, trims, dedupes, and sorts them for `core.Issue`.
- Dependencies must preserve both edge shapes. The local-symphony wrapper maps
  only kind `blocks` into `core.Issue.BlockedBy`, dedupes and sorts IDs, and
  drops self references.
- Timestamps are RFC3339; malformed or empty timestamps become zero time in the
  local-symphony wrapper.

## Error classes to expose

The SDK needs enough structured error information for local-symphony to map:

- command start failure or missing binary -> `CategoryAPIRequest`
- context cancellation or deadline -> `CategoryTimeout` or local canceled
  category at the tool boundary
- stderr containing `not found`, `no such issue`, or `issue does not exist` ->
  `CategoryNotFound`
- stderr containing `permission denied`, `unauthorized`, or `forbidden` ->
  `CategoryAuthFailed`
- stderr containing `invalid status`, `unknown status`, `invalid priority`, or
  `validation failed` -> `CategoryValidation`
- other non-zero bd exit -> `CategoryAPIStatus` with exit code preserved
- malformed JSON, missing required fields, or mismatched `show` ID ->
  `CategoryUnknownPayload`
- unsupported `LinkPR` -> `CategoryUnsupported` and `errors.Is` support for an
  unsupported sentinel

## Beads upstream observations

The public docs for Beads 1.0.4 say programmatic access should use `--json` for
commands such as `bd list --json` and `bd show bd-42 --json`. They also define
`bd ready` as open issues with no active blockers, excluding `in_progress`,
`blocked`, `deferred`, and hooked issues, and state that `bd list --ready` uses
the same blocker-aware semantics.

Relevant documented flags:

- `bd ready`: `--claim`, `--explain`, `-n/--limit`, `--priority`, `--type`,
  `--assignee`, label filters, and metadata filters.
- `bd list`: `--all`, `--status`, `--ready`, `-n/--limit`, `--priority`,
  `--type`, label filters, and timestamp filters.
- `bd show`: JSON output is supported through the global `--json` flag.
- `bd update`: supports `--status` and `--append-notes`.
- `bd close`: supports `--reason`.

Local live JSON was captured with bd 1.0.0, while the website currently
documents 1.0.4. The SDK should keep unknown-field tolerance and fixture tests
for the shapes above rather than overfitting to either exact version.
