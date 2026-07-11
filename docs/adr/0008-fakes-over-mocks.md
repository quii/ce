---
id: 0008
title: Fakes over mocks for out-ports
status: Accepted
scope:
  - internal/adapters/**
  - "**/*_test.go"
enforcement: judgment
---

# 0008: Fakes over mocks for out-ports

## Decision

Every out-port (the event store, the outbox, projections - anything CE depends on externally) gets a real adapter and an in-memory fake, both satisfying the same interface. Domain and use-case code is tested against the fake. Mocks - test doubles that assert on which calls were made - are not used for out-ports.

## Rationale

See [Working without mocks](https://quii.gitbook.io/learn-go-with-tests/testing-fundamentals/working-without-mocks). A fake is a real, working, simplified implementation - it lets a test express behaviour ("given this was stored, when I ask for it back, I get it") instead of implementation detail ("this method was called once with these arguments"), so tests don't break when internals are refactored.

## Consequences

Writing a fake is real work, not a shortcut - it has to actually behave like the thing it stands in for, which is what contract tests (`docs/adr/0009-contract-tests.md`) exist to prove.

## Enforcement

Judgment - a subagent reviewing a new or changed out-port dependency in a test checks whether it's a hand-rolled fake versus a mocking-library-generated double.
