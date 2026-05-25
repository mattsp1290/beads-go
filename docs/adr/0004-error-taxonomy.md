# ADR 0004: SDK error taxonomy

## Status

Accepted

## Context

The SDK wraps a local CLI today, but callers still need stable, structured
errors. Message matching at every consumer would be brittle and would make a
future non-CLI transport harder to introduce.

## Decision

All operation failures that the SDK classifies use:

```go
type Error struct {
	Op     string
	Kind   Kind
	Status int
	Err    error
}
```

`Kind` has eight values:

- `KindNotFound`
- `KindValidation`
- `KindTimeout`
- `KindTransport`
- `KindAuthFailed`
- `KindUnsupported`
- `KindBadResponse`
- `KindExit`

The exported sentinels are:

- `ErrNotFound`
- `ErrUnsupported`
- `ErrValidation`

`errors.Is` matches sentinels by `Kind`, and `errors.As` exposes the full
`*Error` for `Op`, `Status`, and wrapped error inspection.

## Classification Rules

- `context.DeadlineExceeded` and `context.Canceled` become `KindTimeout`.
- Binary startup, look-path, permission, and path errors become
  `KindTransport`.
- CLI stderr containing auth patterns such as `permission denied`,
  `unauthorized`, or `forbidden` becomes `KindAuthFailed`.
- CLI stderr containing not-found patterns such as `not found`, `no such
  issue`, or `issue does not exist` becomes `KindNotFound`.
- CLI stderr containing validation patterns such as `invalid status`, `unknown
  status`, `invalid priority`, or `validation failed` becomes
  `KindValidation`.
- Boundary validation in the SDK also becomes `KindValidation`.
- Malformed JSON, unsupported envelope versions, missing required issue fields,
  and mismatched `Show` IDs become `KindBadResponse`.
- Other non-zero CLI exits become `KindExit`; the process exit code is copied to
  `Error.Status`.

## Future Status Reuse

`Error.Status` currently carries CLI exit status for `KindExit`. A future HTTP
or RPC transport may reuse the same field for remote status codes, but only when
the kind names still preserve the distinction between transport failures,
validation failures, bad payloads, and remote status failures.

## Downstream Mapping

Wrappers can map SDK kinds directly:

- bad response -> unknown payload
- auth failed -> auth failed
- API status / exit -> API status, preserving `Status`
- validation -> validation
- timeout -> timeout
- transport -> API request / transport
- not found -> not found
- unsupported -> unsupported

This is the mapping local-symphony should use in its thin tracker wrapper.

## Consequences

The public taxonomy is intentionally small. Adding new kinds is possible but
should be treated as a compatibility-sensitive change because downstream
wrappers may switch on the current set.

The SDK does not expose raw stderr as a public field. Consumers that need
details can inspect the wrapped `Err`.
