#!/usr/bin/env bash
# Project: beads-go
# Generated: 2026-05-25

set -euo pipefail

if ! command -v bd >/dev/null 2>&1; then
  echo "error: bd CLI is required on PATH" >&2
  exit 1
fi

if [ ! -d ".beads" ]; then
  bd init --non-interactive --skip-agents --skip-hooks --prefix beadsgo --database beadsgo
fi

echo "Creating beads-go v1 task graph..."

# ========================================
# Phase 1: Repository Baseline
# ========================================

SETUP_REPO=$(bd create "Establish repository baseline for beads-go v0.1.0" \
  -d "Verify the repository is clean enough for coordinated work, default branch is main, module path target is github.com/mattsp1290/beads-go, and no unrelated generated files are present. Document any pre-existing drift before implementation tasks begin." \
  -p 0 -l setup --silent)

SETUP_GOMOD=$(bd create "Initialize go.mod for github.com/mattsp1290/beads-go using Go 1.26" \
  -d "Create or verify go.mod with module github.com/mattsp1290/beads-go and go directive 1.26. Keep v1 stdlib-only; go.sum should not appear unless tooling requires it." \
  -p 0 -l setup --silent)
bd dep add "$SETUP_GOMOD" "$SETUP_REPO"

SETUP_LICENSE=$(bd create "Verify MIT LICENSE for 2026 Matt Spurlin" \
  -d "Ensure LICENSE contains the standard MIT License text with copyright year 2026 and holder Matt Spurlin. Update only if absent, stale, or non-standard." \
  -p 1 -l setup --silent)
bd dep add "$SETUP_LICENSE" "$SETUP_REPO"

SETUP_LINT=$(bd create "Add golangci-lint v2 baseline retargeted from local-symphony" \
  -d "Copy the baseline from ~/git/local-symphony/.golangci.yml, set run.go to 1.26, configure goimports local-prefixes as github.com/mattsp1290/beads-go, remove local-symphony-only exclusions such as internal/runtime, and keep formatting checks enabled." \
  -p 1 -l setup --silent)
bd dep add "$SETUP_LINT" "$SETUP_GOMOD"

SETUP_LAYOUT=$(bd create "Create public beads package and docs directory layout" \
  -d "Create beads/ and docs/adr/ directories. Keep v1 as a single public package at github.com/mattsp1290/beads-go/beads with no premature subpackages. Do not add public transport or runner packages." \
  -p 0 -l setup --silent)
bd dep add "$SETUP_LAYOUT" "$SETUP_GOMOD"

RESEARCH_SOURCE=$(bd create "Audit local-symphony bd adapter and bd CLI command contract" \
  -d "Read ~/git/local-symphony/internal/tracker/beads/{beads.go,runner.go,parse.go,errors.go,doc.go} and tests, ~/git/local-symphony/internal/tracker/tracker.go, trackerwrite.go, ~/git/dotfiles/.agents/rules/beads.md, and upstream gastownhall/beads JSON behavior. Record exact command shapes and JSON fixtures needed for the SDK." \
  -p 0 -l research --silent)
bd dep add "$RESEARCH_SOURCE" "$SETUP_REPO"

RESOLVE_NPM_PACKAGE=$(bd create "Resolve bd npm package name and version policy for CI" \
  -d "Inspect gastownhall/beads/npm-package/package.json to confirm package name, bin name, and available version. Decide whether CI pins a known version or floats latest; document the choice in README and CI comments." \
  -p 1 -l research --silent)
bd dep add "$RESOLVE_NPM_PACKAGE" "$RESEARCH_SOURCE"

# ========================================
# Phase 2: Public API Skeleton
# ========================================

API_DOC=$(bd create "Write beads package documentation and compatibility contract" \
  -d "Add beads/doc.go describing package purpose, concurrency safety of *Client, bd data-directory locking caveat, forward compatibility for unknown JSON fields and RawJSON, context-only timeout policy, and absence of logging or telemetry." \
  -p 0 -l api --silent)
bd dep add "$API_DOC" "$SETUP_LAYOUT"

API_CLIENT=$(bd create "Implement Client construction with immutable config and functional options" \
  -d "Add Client, NewClient, WithBinary, WithDataDir, WithActor, and WithListLimit. Validate option values at construction where appropriate. Store immutable config safe for concurrent use. Use bd from PATH by default." \
  -p 0 -l api --silent)
bd dep add "$API_CLIENT" "$API_DOC"

API_LIST_OPTIONS=$(bd create "Implement list option API for state, label, and all filters" \
  -d "Add option types for Ready/List calls including WithState, WithLabel, and WithAll. Keep filters option-based rather than positional so future filters are non-breaking." \
  -p 0 -l api --silent)
bd dep add "$API_LIST_OPTIONS" "$API_CLIENT"

API_NO_RAW=$(bd create "Assert Client exposes no Raw argv escape hatch" \
  -d "Add compile-time/API review coverage ensuring Client only exposes NewClient, options, Ready, List, Show, Close, Comment, and Transition. Do not add Raw(args ...string) or any public CLI-shaped escape hatch." \
  -p 1 -l api --silent)
bd dep add "$API_NO_RAW" "$API_CLIENT"

# ========================================
# Phase 3: Errors and JSON Types
# ========================================

ERR_MODEL=$(bd create "Implement slim SDK Error envelope and sentinels" \
  -d "Add Kind enum with KindNotFound, KindValidation, KindTimeout, KindTransport, KindAuthFailed, KindUnsupported, KindBadResponse, and KindExit. Add Error{Op, Kind, Status, Err}, Error(), Unwrap(), Is(), and sentinels ErrNotFound, ErrUnsupported, ErrValidation." \
  -p 0 -l errors --silent)
bd dep add "$ERR_MODEL" "$API_CLIENT"

ERR_CLASSIFY=$(bd create "Implement exec and validation error classification" \
  -d "Classify context.DeadlineExceeded/context.Canceled as KindTimeout; binary start/look-path/path errors as KindTransport; auth stderr patterns as KindAuthFailed; not-found stderr patterns as KindNotFound; boundary and CLI validation as KindValidation; malformed/unsupported JSON as KindBadResponse; unclassified non-zero exit as KindExit with status." \
  -p 0 -l errors --silent)
bd dep add "$ERR_CLASSIFY" "$ERR_MODEL"

ISSUE_TYPE=$(bd create "Implement normalized Issue and Dependency types with RawJSON" \
  -d "Add Issue fields for ID, Title, Description, Status, Priority *int, Labels, CreatedAt, UpdatedAt, ClosedAt, Dependencies, and RawJSON. Parse RFC3339 timestamps to UTC; missing or malformed timestamps become zero time without failing issue decode. Preserve bd dependency shapes from ready/list and show." \
  -p 0 -l json --silent)
bd dep add "$ISSUE_TYPE" "$ERR_MODEL"

JSON_DECODER=$(bd create "Implement bd JSON decoder for legacy and envelope payloads" \
  -d "Decode legacy arrays/objects and BD_JSON_ENVELOPE=1 payloads shaped as {schema_version,data}. Reject unsupported envelope versions as KindBadResponse. Preserve each issue object exactly in Issue.RawJSON while silently dropping unknown typed fields." \
  -p 0 -l json --silent)
bd dep add "$JSON_DECODER" "$ISSUE_TYPE"

JSON_FIXTURE_TESTS=$(bd create "Add unit tests for issue decoding and RawJSON preservation" \
  -d "Cover Ready, List, and Show payloads with unknown issue fields, both dependency shapes, omitted priority, malformed timestamps, legacy arrays/objects, envelope payloads, unsupported envelope versions, missing required id/status, and wrong show ID." \
  -p 0 -l test --silent)
bd dep add "$JSON_FIXTURE_TESTS" "$JSON_DECODER"

# ========================================
# Phase 4: Exec Transport and Security
# ========================================

SEC_ARGV_HELPERS=$(bd create "Implement positional ID validation and argv separator helpers" \
  -d "Add internal argv/security helpers or package-private equivalents enforcing requireID rejects empty IDs and IDs beginning with '-'. Ensure every public ID-taking method uses a literal -- separator before positional IDs." \
  -p 0 -l security --silent)
bd dep add "$SEC_ARGV_HELPERS" "$API_CLIENT"

RUNNER_SEAM=$(bd create "Implement unexported exec runner seam with G204 documentation" \
  -d "Add unexported runner abstraction over os/exec. Include #nosec G204 and a comment explaining operator-configured binary, internally constructed argv, no shell invocation, requireID validation, and -- separator defense in depth." \
  -p 0 -l transport --silent)
bd dep add "$RUNNER_SEAM" "$SEC_ARGV_HELPERS"

TRANSPORT_EXEC=$(bd create "Implement unexported transport interface and execTransport" \
  -d "Add Client-owned unexported typed transport interface and execTransport as the only built-in v1 transport. Do not export operation enums, request envelopes, transport interfaces, or runner seams." \
  -p 0 -l transport --silent)
bd dep add "$TRANSPORT_EXEC" "$RUNNER_SEAM"

ARGV_REPLAY_FAKE=$(bd create "Build replay-style fake runner for exact argv unit tests" \
  -d "Create unit-test helper beneath execTransport that scripts stdout, stderr, exit status, and expected argv/env/working directory per invocation. It must fail tests on unexpected invocation and support panic-on-call for boundary validation tests." \
  -p 0 -l test --silent)
bd dep add "$ARGV_REPLAY_FAKE" "$TRANSPORT_EXEC"

SECURITY_TESTS=$(bd create "Add boundary validation tests for every ID-taking method" \
  -d "Verify Show, Close, Comment, and Transition reject empty IDs and id '-rf' with KindValidation without invoking the runner. Tests must use a fake runner that panics if called." \
  -p 0 -l test --silent)
bd dep add "$SECURITY_TESTS" "$ARGV_REPLAY_FAKE"

# ========================================
# Phase 5: SDK Methods
# ========================================

METHOD_READY=$(bd create "Implement Client.Ready through bd ready JSON output" \
  -d "Build exact argv for bd ready with --json, configured --db, --actor, list limit, labels where supported, and optional filters. Decode []Issue via shared decoder. Return non-nil empty slices." \
  -p 0 -l sdk --silent)
bd dep add "$METHOD_READY" "$TRANSPORT_EXEC"
bd dep add "$METHOD_READY" "$JSON_DECODER"
bd dep add "$METHOD_READY" "$API_LIST_OPTIONS"

METHOD_LIST=$(bd create "Implement Client.List with state, label, all, and limit options" \
  -d "Build exact argv for bd list including --json, --all when requested, --status/--state filter as verified from bd contract, labels, configured --db and --actor, and list limit. Decode []Issue via shared decoder." \
  -p 0 -l sdk --silent)
bd dep add "$METHOD_LIST" "$METHOD_READY"

METHOD_SHOW=$(bd create "Implement Client.Show with ErrNotFound and echoed-ID validation" \
  -d "Validate id, invoke bd show with --json and -- before the id, decode a single Issue, map not-found to ErrNotFound/KindNotFound, and reject malformed payloads or returned IDs that do not match the requested ID as KindBadResponse." \
  -p 0 -l sdk --silent)
bd dep add "$METHOD_SHOW" "$TRANSPORT_EXEC"
bd dep add "$METHOD_SHOW" "$JSON_DECODER"
bd dep add "$METHOD_SHOW" "$SEC_ARGV_HELPERS"

METHOD_CLOSE=$(bd create "Implement Client.Close with optional reason" \
  -d "Validate id, invoke bd close with -- before id, include --reason only when the reason is non-empty after trimming, and map exec failures through the shared classifier." \
  -p 0 -l sdk --silent)
bd dep add "$METHOD_CLOSE" "$TRANSPORT_EXEC"
bd dep add "$METHOD_CLOSE" "$SEC_ARGV_HELPERS"

METHOD_COMMENT=$(bd create "Implement Client.Comment using bd notes/comment append behavior" \
  -d "Validate id and non-empty body according to boundary rules, invoke the verified bd command for appending notes/comments with --append-notes or equivalent current CLI flag, and place -- before the positional id." \
  -p 0 -l sdk --silent)
bd dep add "$METHOD_COMMENT" "$TRANSPORT_EXEC"
bd dep add "$METHOD_COMMENT" "$SEC_ARGV_HELPERS"
bd dep add "$METHOD_COMMENT" "$RESEARCH_SOURCE"

METHOD_TRANSITION=$(bd create "Implement Client.Transition for status changes" \
  -d "Validate id and state, invoke bd update or equivalent verified command with --status=<state> and -- before the id, and classify validation stderr as KindValidation." \
  -p 0 -l sdk --silent)
bd dep add "$METHOD_TRANSITION" "$TRANSPORT_EXEC"
bd dep add "$METHOD_TRANSITION" "$SEC_ARGV_HELPERS"
bd dep add "$METHOD_TRANSITION" "$RESEARCH_SOURCE"

METHOD_UNIT_TESTS=$(bd create "Add replay-runner unit tests for all SDK methods" \
  -d "Cover Ready, List, Show, Close, Comment, and Transition success and failure paths. Assert exact argv placement for --json, --db=, --actor=, --, -n/list limit, --all, --reason=, --append-notes=, and --status=. Cover timeout, transport error, auth stderr, validation stderr, not-found stderr, bad JSON, unsupported kind, and exit status." \
  -p 0 -l test --silent)
bd dep add "$METHOD_UNIT_TESTS" "$METHOD_READY"
bd dep add "$METHOD_UNIT_TESTS" "$METHOD_LIST"
bd dep add "$METHOD_UNIT_TESTS" "$METHOD_SHOW"
bd dep add "$METHOD_UNIT_TESTS" "$METHOD_CLOSE"
bd dep add "$METHOD_UNIT_TESTS" "$METHOD_COMMENT"
bd dep add "$METHOD_UNIT_TESTS" "$METHOD_TRANSITION"
bd dep add "$METHOD_UNIT_TESTS" "$ARGV_REPLAY_FAKE"
bd dep add "$METHOD_UNIT_TESTS" "$SECURITY_TESTS"

NO_TELEMETRY_TEST=$(bd create "Verify library imports no logging or telemetry packages" \
  -d "Add static test or review gate confirming beads-go does not import log, log/slog, OpenTelemetry packages, or any third-party dependency. Callers own logging and telemetry." \
  -p 1 -l test --silent)
bd dep add "$NO_TELEMETRY_TEST" "$METHOD_UNIT_TESTS"

# ========================================
# Phase 6: Real bd Integration
# ========================================

INTEGRATION_HARNESS=$(bd create "Add integration test harness for real bd binary" \
  -d "Create tests behind the integration build tag. Each test must use a temp dir, run git init, run bd init --non-interactive --skip-agents --skip-hooks --prefix bgotest --database bgotest, and point SDK calls at the temp repo or .beads data directory." \
  -p 0 -l integration --silent)
bd dep add "$INTEGRATION_HARNESS" "$METHOD_UNIT_TESTS"
bd dep add "$INTEGRATION_HARNESS" "$RESOLVE_NPM_PACKAGE"

INTEGRATION_SUCCESS=$(bd create "Exercise full v1 method set against real bd binary" \
  -d "Add integration coverage for Ready, List, Show, Close, Comment, and Transition using real bd state. Verify end-to-end success paths, non-nil slices, decoded fields, dependencies where feasible, notes/comments, and status transitions." \
  -p 0 -l integration --silent)
bd dep add "$INTEGRATION_SUCCESS" "$INTEGRATION_HARNESS"

INTEGRATION_ERRORS=$(bd create "Cover stable real-binary stderr error classes" \
  -d "Add integration tests for bd show with a bogus ID producing KindNotFound and at least one stable CLI validation failure producing KindValidation if available. Keep bad-response and timeout coverage in unit tests." \
  -p 1 -l integration --silent)
bd dep add "$INTEGRATION_ERRORS" "$INTEGRATION_HARNESS"

INTEGRATION_ENVELOPE=$(bd create "Cover BD_JSON_ENVELOPE=1 with integration or targeted fixtures" \
  -d "Ensure envelope JSON shape is covered. Prefer an integration matrix or targeted integration case with BD_JSON_ENVELOPE=1; if real bd behavior is unstable, retain unit fixtures and document the integration limitation." \
  -p 1 -l integration --silent)
bd dep add "$INTEGRATION_ENVELOPE" "$INTEGRATION_HARNESS"
bd dep add "$INTEGRATION_ENVELOPE" "$JSON_FIXTURE_TESTS"

# ========================================
# Phase 7: CI
# ========================================

CI_UNIT=$(bd create "Add GitHub Actions unit quality workflow for Go 1.26" \
  -d "Create .github/workflows/ci.yml with a unit job using actions/setup-go for 1.26.x or go-version-file. Run go mod tidy --diff, go vet ./..., golangci-lint run, formatting verification, and go test -race ./...." \
  -p 0 -l ci --silent)
bd dep add "$CI_UNIT" "$SETUP_LINT"
bd dep add "$CI_UNIT" "$METHOD_UNIT_TESTS"
bd dep add "$CI_UNIT" "$NO_TELEMETRY_TEST"

CI_BD_BINARY=$(bd create "Add GitHub Actions integration job installing bd from npm" \
  -d "Add a separate integration job that sets up Node, runs npm install -g for the resolved beads package/version, verifies bd --version, and runs go test -tags=integration ./.... Document whether npm dependency is pinned or floating." \
  -p 0 -l ci --silent)
bd dep add "$CI_BD_BINARY" "$CI_UNIT"
bd dep add "$CI_BD_BINARY" "$INTEGRATION_SUCCESS"
bd dep add "$CI_BD_BINARY" "$INTEGRATION_ERRORS"
bd dep add "$CI_BD_BINARY" "$INTEGRATION_ENVELOPE"

# ========================================
# Phase 8: local-symphony Acceptance Gate
# ========================================

LOCAL_SYMPHONY_AUDIT=$(bd create "Audit local-symphony replacement requirements before wrapper work" \
  -d "In ~/git/local-symphony, inspect internal/tracker/beads tests, runtime wiring, tracker interfaces, and worker trackerwrite usage. Identify exact wrapper methods and tests that must remain behavior-compatible when internal exec adapter is replaced by github.com/mattsp1290/beads-go/beads." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_SYMPHONY_AUDIT" "$METHOD_UNIT_TESTS"

LOCAL_REPLACE=$(bd create "Add temporary local-symphony replace to beads-go" \
  -d "In ~/git/local-symphony, add a temporary replace github.com/mattsp1290/beads-go => ../beads-go while validating the replacement. Do not commit the replace unless the acceptance workflow explicitly keeps it on a temporary branch." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_REPLACE" "$LOCAL_SYMPHONY_AUDIT"
bd dep add "$LOCAL_REPLACE" "$CI_UNIT"

LOCAL_WRAPPER=$(bd create "Replace local-symphony bd exec adapter with beads-go wrapper" \
  -d "Preserve package path internal/tracker/beads if it minimizes churn. Wrap *beads.Client to satisfy local tracker.Tracker. Do not re-export beads-go transport details. Keep runtime import changes minimal." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_WRAPPER" "$LOCAL_REPLACE"
bd dep add "$LOCAL_WRAPPER" "$METHOD_READY"
bd dep add "$LOCAL_WRAPPER" "$METHOD_LIST"
bd dep add "$LOCAL_WRAPPER" "$METHOD_SHOW"
bd dep add "$LOCAL_WRAPPER" "$METHOD_CLOSE"
bd dep add "$LOCAL_WRAPPER" "$METHOD_COMMENT"
bd dep add "$LOCAL_WRAPPER" "$METHOD_TRANSITION"

LOCAL_READERS=$(bd create "Preserve local-symphony reader semantics on beads-go wrapper" \
  -d "Ensure FetchCandidates calls SDK Ready, filters by activeStates client-side, returns non-nil empty slices, and still invokes Ready when activeStates is empty. Ensure FetchByStates short-circuits empty/nil/all-empty states, otherwise fans out List WithState + WithAll per state and dedupes by ID first-state-wins." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_READERS" "$LOCAL_WRAPPER"

LOCAL_FETCH_IDS=$(bd create "Preserve local-symphony FetchStatesByIDs semantics" \
  -d "Ensure empty input returns empty map without invoking bd, IDs are deduped, empty IDs are skipped, ErrNotFound is tolerated by omission, hard errors return partial map, and mismatched echoed IDs become local CategoryUnknownPayload." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_FETCH_IDS" "$LOCAL_WRAPPER"

LOCAL_MAPPING=$(bd create "Preserve local-symphony issue mapping behavior" \
  -d "Map priority nil and 0..4 to core.Priority including unset/backlog; lowercase, trim, dedupe, and sort labels; map dependency kind blocks to core.Issue.BlockedBy across both bd JSON shapes; drop self references; keep timestamps UTC and malformed timestamps zero." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_MAPPING" "$LOCAL_WRAPPER"

LOCAL_WRITERS=$(bd create "Preserve local-symphony writer behavior on beads-go wrapper" \
  -d "Close trims reason and omits --reason for empty reason. Comment and Transition compose SDK methods while preserving current local behavior. LinkPR remains unsupported. RateLimit returns zero tracker.RateLimitSnapshot." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_WRITERS" "$LOCAL_WRAPPER"

LOCAL_ERROR_MAP=$(bd create "Map beads-go errors into local-symphony tracker categories" \
  -d "Map KindBadResponse to CategoryUnknownPayload, KindAuthFailed to CategoryAuthFailed, KindExit to CategoryAPIStatus, and validation, timeout, transport, not-found, and unsupported to their existing local equivalents. Preserve wrapping enough for errors.Is where expected." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_ERROR_MAP" "$LOCAL_WRAPPER"

LOCAL_TESTS=$(bd create "Update and run local-symphony tests for beads-go replacement" \
  -d "Update local-symphony tests only as needed for the wrapper. Run the relevant internal/tracker/beads tests, tracker conformance tests, and broader local-symphony test suite required by the existing project. All updated tests must pass with the temporary replace." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_TESTS" "$LOCAL_READERS"
bd dep add "$LOCAL_TESTS" "$LOCAL_FETCH_IDS"
bd dep add "$LOCAL_TESTS" "$LOCAL_MAPPING"
bd dep add "$LOCAL_TESTS" "$LOCAL_WRITERS"
bd dep add "$LOCAL_TESTS" "$LOCAL_ERROR_MAP"

LOCAL_PR=$(bd create "Open local-symphony acceptance PR for beads-go replacement" \
  -d "Create a PR against ~/git/local-symphony replacing the existing bd exec adapter with the beads-go wrapper. Include test results, note the temporary replace requirement before v0.1.0, and ensure the PR is ready to switch to the tagged module after release." \
  -p 0 -l consumer --silent)
bd dep add "$LOCAL_PR" "$LOCAL_TESTS"

# ========================================
# Phase 9: Documentation and ADRs
# ========================================

ADR_SCOPE=$(bd create "Write ADR 0001 for v1 scope and flat Client API" \
  -d "Document why v1 exposes one beads package, concrete Client, six methods, option-based list filters, no Tracker interface, no Raw escape hatch, and deferred Create/Update/Dep/Graph/Init/Remember/Prime/Dolt/RateLimit surfaces." \
  -p 1 -l docs --silent)
bd dep add "$ADR_SCOPE" "$API_NO_RAW"

ADR_JSON=$(bd create "Write ADR 0002 for JSON shape, Issue normalization, and RawJSON" \
  -d "Document legacy and envelope JSON support, unsupported envelope handling, timestamp zero-time policy, Priority *int, label pass-through, dependency shape preservation, and per-issue RawJSON forward compatibility." \
  -p 1 -l docs --silent)
bd dep add "$ADR_JSON" "$JSON_FIXTURE_TESTS"

ADR_SECURITY=$(bd create "Write ADR 0003 for exec security contract" \
  -d "Document requireID empty and dash-prefix rejection, literal -- separator before positional IDs, #nosec G204 rationale, no shell invocation, operator-configured binary assumptions, and defense-in-depth relationship between both controls." \
  -p 1 -l docs --silent)
bd dep add "$ADR_SECURITY" "$SECURITY_TESTS"

ADR_ERRORS=$(bd create "Write ADR 0004 for SDK error taxonomy" \
  -d "Document the eight Kind values, sentinels, classification rules, future HTTP status reuse, and how downstream wrappers can map bad response, auth failed, API status, validation, timeout, transport, not-found, and unsupported cases." \
  -p 1 -l docs --silent)
bd dep add "$ADR_ERRORS" "$ERR_CLASSIFY"

ADR_LOCAL_SYMPHONY=$(bd create "Write ADR 0005 for local-symphony adapter contract" \
  -d "Document why beads-go does not export local-symphony's Tracker triad and how local-symphony wraps *beads.Client to preserve FetchCandidates, FetchByStates, FetchStatesByIDs, writer behavior, RateLimit, LinkPR unsupported, and error mapping." \
  -p 1 -l docs --silent)
bd dep add "$ADR_LOCAL_SYMPHONY" "$LOCAL_SYMPHONY_AUDIT"
bd dep add "$LOCAL_PR" "$ADR_LOCAL_SYMPHONY"
bd dep add "$LOCAL_ERROR_MAP" "$ADR_ERRORS"

README_DOCS=$(bd create "Write README with install, example, bd version, and consumer link" \
  -d "README must open with one sentence describing the SDK, include a concise NewClient/Ready/Close example, mention supported/tested bd npm package and install command, link to local-symphony as the first consumer, and include the pre-v1 semver caveat." \
  -p 1 -l docs --silent)
bd dep add "$README_DOCS" "$METHOD_UNIT_TESTS"
bd dep add "$README_DOCS" "$RESOLVE_NPM_PACKAGE"

CHANGELOG=$(bd create "Add hand-curated CHANGELOG starting at Unreleased" \
  -d "Create CHANGELOG.md with an Unreleased section for v0.1.0 work. Include a pre-v1 note that breaking changes may occur under v0.x and every future breaking change must include a migration note." \
  -p 1 -l docs --silent)
bd dep add "$CHANGELOG" "$SETUP_REPO"

RELEASE_CHECKLIST=$(bd create "Add v0.1.0 release checklist" \
  -d "Document release steps: run full CI locally, ensure local-symphony replacement passes with temporary replace, tag v0.1.0, remove temporary replace after tag, verify pkg.go.dev, and update CHANGELOG." \
  -p 2 -l docs --silent)
bd dep add "$RELEASE_CHECKLIST" "$README_DOCS"
bd dep add "$RELEASE_CHECKLIST" "$CHANGELOG"
bd dep add "$RELEASE_CHECKLIST" "$CI_BD_BINARY"

# ========================================
# Phase 10: Final Verification and Release Readiness
# ========================================

VERIFY_LOCAL=$(bd create "Run full beads-go local verification suite" \
  -d "Run go mod tidy --diff, gofmt/goimports or golangci-lint fmt --diff, go vet ./..., golangci-lint run, go test -race ./..., and go test -tags=integration ./... with bd installed. Record any skipped integration assumptions explicitly." \
  -p 0 -l verify --silent)
bd dep add "$VERIFY_LOCAL" "$CI_BD_BINARY"
bd dep add "$VERIFY_LOCAL" "$README_DOCS"
bd dep add "$VERIFY_LOCAL" "$ADR_SCOPE"
bd dep add "$VERIFY_LOCAL" "$ADR_JSON"
bd dep add "$VERIFY_LOCAL" "$ADR_SECURITY"
bd dep add "$VERIFY_LOCAL" "$ADR_ERRORS"
bd dep add "$VERIFY_LOCAL" "$RELEASE_CHECKLIST"

VERIFY_API=$(bd create "Review exported API before v0.1.0 tag" \
  -d "Run go doc or equivalent API inspection. Confirm only intended package symbols are exported, transport and runner seams remain hidden, no Raw escape hatch exists, no public operation enum leaked, and docs mention pre-v1 compatibility." \
  -p 0 -l verify --silent)
bd dep add "$VERIFY_API" "$VERIFY_LOCAL"
bd dep add "$VERIFY_API" "$API_NO_RAW"

VERIFY_CONSUMER=$(bd create "Confirm local-symphony acceptance gate passes" \
  -d "Verify the local-symphony replacement PR exists, uses github.com/mattsp1290/beads-go with temporary local replace before tag, preserves behavior, and has passing tests. This is the v1 success gate." \
  -p 0 -l verify --silent)
bd dep add "$VERIFY_CONSUMER" "$LOCAL_PR"
bd dep add "$VERIFY_CONSUMER" "$VERIFY_API"

TAG_V010=$(bd create "Tag beads-go v0.1.0 and verify pkg.go.dev" \
  -d "After all verification and local-symphony acceptance pass, tag v0.1.0, push the tag, verify pkg.go.dev indexes github.com/mattsp1290/beads-go/beads, update local-symphony to remove the temporary replace, and record the final release state in CHANGELOG." \
  -p 1 -l release --silent)
bd dep add "$TAG_V010" "$VERIFY_CONSUMER"

echo ""
echo "Beads graph created."
echo "Next commands:"
echo "  bd ready"
echo "  bd list --all"
