---
id: 0002
title: Tests only exercise public APIs
status: Accepted
scope:
  - "**/*_test.go"
enforcement: mechanical
---

# 0002: Tests only exercise public APIs

## Decision

Every test file is in an external test package - `package mypkg_test`, never `package mypkg`. No exceptions.

## Rationale

A test in an external package can never reach into unexported internals. If something feels untestable without breaking that rule, the design or the public API needs to change - the test doesn't get special access to work around it.

## Consequences

Some internal behaviour can only be verified indirectly, through the public API. That's the intended pressure, not a workaround to route around.

## Enforcement

Mechanical - the `testpackage` linter rule. CI fails on any `_test.go` file that isn't in a `_test` package.
