---
id: 0014
title: Outside-in TDD
status: Accepted
scope: []
enforcement: process
---

# 0014: Outside-in TDD

## Decision

The first test for a new piece of behaviour is at the boundary - typically an in-port/use-case test, sometimes an HTTP handler test - not a unit test for some internal type that doesn't have a caller yet. Let the failing test tell you what needs to exist next. Red, green, refactor, in small steps.

## Rationale

Designing the internals up front and backfilling tests afterwards produces tests that confirm the implementation rather than ones that would have caught a wrong design. This project follows [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests)'s approach throughout, not just for this rule.

## Consequences

Internal types and functions come into existence because a boundary-level test demanded them, not ahead of that.

## Enforcement

None directly, and deliberately not a subagent check against the diff. A diff is a snapshot of the end state - code written test-first and code written implementation-first then backfilled with a test can produce an identical diff, so there's no structural smell that would actually distinguish them. This is a decision record for the coding agent to read before writing code (`docs/story-process.md`), not something reviewed after the fact.

The guarantees this practice is meant to produce are covered by ADRs that check something a diff actually shows: `docs/adr/0013-implement-only-the-current-test.md` (scope - is there code beyond what the current test demands) and `docs/adr/0020-mutation-testing.md` (mechanically catches code with no test coverage, regardless of the order it was written in).
