---
id: 0005
title: No new third-party dependencies without flagging it first
status: Accepted
scope:
  - go.mod
enforcement: mechanical
---

# 0005: No new third-party dependencies without flagging it first

## Decision

This project is deliberately standard-library-only. Reaching for a third-party package to solve something the standard library already handles is a signal to stop and check, not to `go get` and move on.

## Rationale

Consistent with the "no frameworks whatsoever" tech stack choice - a dependency added quietly is a decision nobody actually made.

## Consequences

Anything that seems to need a dependency gets flagged for a conversation before it's added, not after.

## Enforcement

Mechanical - a test that fails if `go.mod` contains any `require` beyond an explicit allowlist, which starts empty. `go get`-ing something new breaks the build immediately rather than depending on anyone remembering to flag it.

Scoped to the `require` block only - `go.mod`'s `tool` directive (build-time tooling like `mage`, `golangci-lint`, `go-mutesting`) is intentionally out of scope. Tooling never ships in the production image, and any addition already shows up as a reviewable diff in `go.mod`, so it doesn't need the same mechanical gate as a runtime dependency.
