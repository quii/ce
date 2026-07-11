---
id: 0016
title: Don't loosen a test to make it pass
status: Accepted
scope:
  - "**/*_test.go"
enforcement: judgment
---

# 0016: Don't loosen a test to make it pass

## Decision

If a test fails, either the code is wrong or the test's expectation was wrong - fix whichever one it actually is. Widening an assertion, deleting a case, or softening what's being checked just to get back to green is never acceptable, even as a temporary measure.

## Rationale

This is stated bluntly because it's a documented, common failure mode for agents specifically - not something a human on this project would likely do by habit.

## Consequences

A genuinely wrong test expectation still gets changed - but that's a distinct, deliberate act with its own justification, not a side effect of chasing a green build.

## Enforcement

Judgment, and explicitly the hardest of these to get right - expect false positives. A subagent diffs test files specifically for weakened assertions, deleted cases, or widened tolerances, and treats any of those as requiring justification rather than silent passage.
