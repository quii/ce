---
id: 0015
title: UTC always for timestamps
status: Accepted
scope:
  - "**/*.go"
enforcement: judgment
---

# 0015: UTC always for timestamps

## Decision

Every `time.Time` is UTC, always - fixed at the point where it's created (the handler stamping `OccurredAt`, a real or fake `Clock`), not converted later.

## Rationale

Two timestamps representing the same instant can still fail an equality check if one carries a UTC location and the other a local one - a subtle, real source of non-determinism that looks like a formatting detail but is actually a correctness one, and exactly the kind of thing `docs/adr/0021-no-flaky-tests.md` doesn't tolerate.

## Consequences

None beyond discipline at the small number of places time actually gets created.

## Enforcement

Judgment - a subagent checks new `time.Time` creation points (`Clock` implementations, command construction stamping `OccurredAt`) for an explicit or implicit non-UTC location.
