# ADR 0003: exec security contract

## Status

Accepted

## Context

The v0.1.0 SDK talks to bd by executing the local `bd` binary. That gives Go
callers a useful API before bd has a stable RPC transport, but it means argv
construction is a security boundary.

## Decision

The SDK invokes bd with `exec.CommandContext` and never through a shell.

The bd binary is operator-configured:

- Default: `bd` resolved from `PATH`.
- Optional: explicit executable name or path through `WithBinary`.

The SDK constructs argv internally. Callers can pass values such as issue IDs,
states, labels, reasons, and comment bodies, but they cannot pass raw argv.

## Positional ID Rules

Every public method that accepts a positional issue ID must call `requireID`.
That helper:

- Trims surrounding whitespace.
- Rejects empty IDs.
- Rejects IDs beginning with `-`.

Every public method that places an issue ID after flags must use
`appendPositionalID`, which inserts a literal `--` before the ID.

Examples:

```text
bd --json show -- beadsgo-123
bd --json close --reason=done -- beadsgo-123
bd --json update --append-notes=note -- beadsgo-123
bd --json update --status=closed -- beadsgo-123
```

## G204 Rationale

`execRunner` carries a `#nosec G204` annotation because gosec cannot know the
package-level contract:

- The binary is selected by the operator or defaults to `bd` from `PATH`.
- No shell is invoked.
- Subcommands and flags are constructed by SDK code.
- Positional IDs are validated before they become argv.
- A literal `--` separates flags from positional IDs.

This is an intentional command execution boundary, not string interpolation.

## Defense In Depth

The dash-prefix rejection and the `--` separator are both required.

The `--` separator prevents bd's flag parser from treating a valid positional
ID as a flag. The dash-prefix rejection prevents the SDK from even attempting to
send an option-shaped ID if a future call site forgets the separator.

Boundary tests use a panic-on-call runner to prove invalid IDs fail before any
subprocess invocation.

## Consequences

The SDK does not expose `Raw(args ...string)` or any public runner/transport
escape hatch.

Adding a future RPC transport should not require emulating arbitrary CLI argv,
because the public API is method-shaped rather than command-shaped.
