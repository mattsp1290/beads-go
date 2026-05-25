# Changelog

All notable changes to this project are documented here.

## v0.1.0 - 2026-05-25

### Added

- Initial `github.com/mattsp1290/beads-go/beads` package skeleton.
- `Client` construction with functional options for bd binary, data directory,
  actor, and list limit.
- SDK error envelope and sentinels for not-found, validation, and unsupported
  cases.
- Exec runner and transport seams for invoking bd without a shell.
- Public issue and dependency types with per-issue `RawJSON` preservation.
- Legacy and `BD_JSON_ENVELOPE=1` JSON decoding.
- `Ready`, `List`, `Show`, `Close`, `Comment`, and `Transition` client
  methods.
- Unit test replay runner for exact argv and error classification coverage.
- Research notes for the bd command contract and local-symphony replacement
  wrapper.
- GitHub Actions unit and integration jobs. The integration job installs the
  pinned npm package `@beads/bd@1.0.4`.
- ADRs for v1 scope, JSON shape, exec security, error taxonomy, and the
  local-symphony adapter contract.
- v0.1.0 release checklist.
- local-symphony acceptance PR using a temporary local replace until this tag is
  available.

### Compatibility

- This project is pre-v1. Breaking changes may occur under v0.x.
- Every future breaking change must include a migration note in this changelog.
