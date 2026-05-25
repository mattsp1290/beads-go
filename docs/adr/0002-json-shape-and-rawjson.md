# ADR 0002: JSON shape, Issue normalization, and RawJSON

## Status

Accepted

## Context

bd JSON output is the SDK's wire format in v0.1.0. Current bd commands emit
legacy arrays and objects, and newer bd builds may emit an envelope when
`BD_JSON_ENVELOPE=1` is enabled:

```json
{"schema_version":1,"data":[...]}
```

The SDK needs typed fields for normal use while staying forward-compatible with
bd adding new fields.

## Decision

The decoder supports:

- Legacy arrays: `[{"id":"x","status":"open"}]`
- Legacy objects: `{"id":"x","status":"open"}`
- Envelope payloads with `schema_version: 1` and `data` containing an array,
  object, or null.

Unsupported envelope versions are rejected as `KindBadResponse`.

`Issue` contains:

- `ID`
- `Title`
- `Description`
- `Status`
- `Priority *int`
- `Labels []string`
- `CreatedAt`, `UpdatedAt`, `ClosedAt`
- `Dependencies []Dependency`
- `RawJSON json.RawMessage`

## Normalization

Timestamps are parsed as RFC3339 and normalized to UTC. Missing or malformed
timestamps become zero `time.Time` values without failing the issue decode.

`Priority` is a pointer so omitted priority can be distinguished from explicit
priority `0`.

Labels pass through exactly as bd sends them. Consumers that need
case-normalized or sorted labels, such as local-symphony, should normalize in
their own wrapper.

Dependencies preserve both bd shapes:

- ready/list edge shape: `issue_id`, `depends_on_id`, `type`
- show embedded shape: `id`, `dependency_type`

The SDK does not interpret dependency kind. Consumers decide which kinds matter
for their workflow.

## RawJSON

Each `Issue.RawJSON` stores the original JSON object for that issue, not the
enclosing list or envelope. Unknown typed fields are ignored, but retained in
`RawJSON`.

This lets callers inspect new bd fields before the SDK exposes first-class typed
accessors.

## Required Field Policy

Decoded issues must include non-empty `id` and `status`. Missing either field is
reported as `KindBadResponse`.

`Client.Show` additionally rejects a non-empty returned ID that differs from the
requested ID as `KindBadResponse`.

## Consequences

Adding a typed field to `Issue` is non-breaking because unknown JSON fields are
already preserved in `RawJSON`.

Renaming or retyping existing public fields is breaking and must be called out
in the changelog with a migration note while the module is still v0.x.
