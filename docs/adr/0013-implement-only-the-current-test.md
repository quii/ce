---
id: 0013
title: Implement only what the current failing test requires
status: Accepted
scope:
  - "**/*.go"
enforcement: judgment
---

# 0013: Implement only what the current failing test requires

## Decision

Implement only what the current failing test requires - don't build ahead of it in anticipation of where the story is probably going. If the next step is obvious, write the next test and take it, rather than writing the code for it early.

## Rationale

This is the one part of ping-pong pairing worth keeping without the handoff machinery: the discipline of only ever seeing the current failing test and its output, not the whole intended solution, is what stops an agent (or a person) from surging ahead and skipping the incremental design pressure TDD is supposed to apply.

## Consequences

Progress can feel slower in the moment - the payoff is that everything that gets built is actually demanded by a test, not assumed to be needed.

## Enforcement

Judgment - a subagent compares a diff's production code against the story's current test coverage, flagging code that isn't demanded by any test yet. This complements mutation testing (`docs/adr/0020-mutation-testing.md`), which catches the same problem mechanically but only after the fact, on the next commit's diff.
