---
id: 0009
title: Contract tests for fakes
status: Accepted
scope:
  - internal/adapters/**
enforcement: judgment
---

# 0009: Contract tests for fakes

## Decision

Each out-port has one shared contract test suite, written against the interface rather than any specific implementation, run twice: once against the fake, once against the real adapter (e.g. real Postgres, brought up via testcontainers for the run).

## Rationale

A fake is only useful if it behaves the same as the real thing it stands in for. If the fake and the real adapter ever disagree, the contract test catches it - this is what makes it safe to write and test domain code entirely against fakes and still trust it against the real thing.

## Consequences

A new out-port isn't done when it has a fake and an adapter - it's done when both pass the same shared contract test suite.

## Enforcement

Judgment - a subagent checks whether a new fake/adapter pair has a corresponding shared contract test exercising both, not just independent unit coverage of each.
