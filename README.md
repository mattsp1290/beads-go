# beads-go

beads-go is a small Go SDK for the `bd` (beads) task-graph CLI.

It shells out to `bd` and gives Go callers a typed API for ready work, lists,
single issue lookup, close, comment, and transition operations.

## bd Version

The SDK targets the `bd` binary distributed by the npm package `@beads/bd`,
which exposes the `bd` executable.

For reproducible CI, install a pinned package version:

```sh
npm install -g @beads/bd@1.0.4
bd version
```

Do not float `latest` in CI. Update the pinned version deliberately after
verifying the JSON contract used by this SDK.

## Install

```sh
go get github.com/mattsp1290/beads-go
```

## Example

```go
package main

import (
	"context"
	"log"

	"github.com/mattsp1290/beads-go/beads"
)

func main() {
	ctx := context.Background()

	client, err := beads.NewClient(
		beads.WithDataDir("/path/to/repo/.beads"),
		beads.WithActor("orchestrator"),
		beads.WithListLimit(50),
	)
	if err != nil {
		log.Fatal(err)
	}

	issues, err := client.Ready(ctx, beads.WithAll())
	if err != nil {
		log.Fatal(err)
	}
	if len(issues) == 0 {
		return
	}

	if err := client.Close(ctx, issues[0].ID, "completed by automation"); err != nil {
		log.Fatal(err)
	}
}
```

## Consumers

The first consumer is
[local-symphony](https://github.com/mattsp1290/local-symphony), which wraps
`*beads.Client` in its internal tracker adapter.

## Compatibility

This module is pre-v1. Breaking changes may occur under v0.x, and every future
breaking change must include a migration note in `CHANGELOG.md`.
