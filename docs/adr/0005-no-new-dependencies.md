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

Favour the standard library. A third-party dependency earns its place only when there's no legitimate standard-library way to do the job - not because a package is more convenient or saves a few lines. Even then, the dependency has to stay additive: it solves its one problem and sits behind the existing architectural seams (hexagonal layering, thin handlers, specifications/drivers), rather than reshaping how the rest of the codebase is written or organised. Reaching for a package to solve something the standard library already handles is a signal to stop and check, not to `go get` and move on.

## Rationale

Consistent with the "no frameworks whatsoever" tech stack choice - a dependency added quietly is a decision nobody actually made, and a dependency that reshapes the codebase around itself is a much bigger commitment than the one line in `go.mod` suggests.

## Consequences

Anything that seems to need a dependency gets flagged for a conversation before it's added, not after: what's the stdlib gap it fills, and does adopting it change how the surrounding code is organised. `docs/adr/0024-openapi-spec-first-with-oapi-codegen.md` is a worked example that clears both bars - OpenAPI-driven server/client generation has no standard-library equivalent, and the generated code sits behind the existing seams (handlers still just translate request/response, drivers still just implement the in-port) rather than dictating them.

## Enforcement

Mechanical - a test that fails if `go.mod` contains any `require` beyond an explicit allowlist, which starts empty. `go get`-ing something new breaks the build immediately rather than depending on anyone remembering to flag it.

Scoped to the `require` block only - `go.mod`'s `tool` directive (build-time tooling like `mage`, `golangci-lint`, `go-mutesting`) is intentionally out of scope. Tooling never ships in the production image, and any addition already shows up as a reviewable diff in `go.mod`, so it doesn't need the same mechanical gate as a runtime dependency.
