---
id: 0004
title: File length
status: Accepted
scope:
  - "**/*.go"
enforcement: mechanical
---

# 0004: File length

## Decision

Every `.go` file is capped at 250 lines. `_test.go` files are exempt - table-driven tests with a lot of cases legitimately run long, and splitting a test file for its own sake doesn't buy anything a shorter production file split does.

## Rationale

The same reasoning as splitting these docs into small, single-concern files: a file that's grown past the limit is usually doing more than one job, and forcing the split benefits a human and an agent working in it equally.

## Consequences

Treat 250 as an ambitious starting point, not a permanent constant - if it proves unworkable in practice it'll be revised, but the default response to hitting it is to split the file, not raise the number.

## Enforcement

Mechanical - `revive`'s `file-length-limit` rule via `golangci-lint run ./...`. The `_test.go` exemption is a golangci-lint `exclude-rules` entry scoped to that rule specifically (matched on both path and the rule's own message, not a blanket exemption from every revive rule for test files).
