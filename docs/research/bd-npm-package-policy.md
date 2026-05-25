# bd npm package policy

Date: 2026-05-25

Sources checked:

- `https://raw.githubusercontent.com/gastownhall/beads/main/npm-package/package.json`
- `npm view @beads/bd name version bin --json`

Findings:

- Package name: `@beads/bd`
- Binary name: `bd`
- Upstream `main` package.json version observed: `1.0.3`
- npm registry version observed: `1.0.4`

Policy:

- CI should install `@beads/bd@1.0.4`, not `@beads/bd@latest`.
- The package must be deliberately bumped after verifying the SDK JSON contract
  against the new bd version.
- CI comments should state:
  `Pinned for reproducible bd JSON behavior; bump only after SDK integration fixtures pass against the new @beads/bd version.`
