---
id: 0020
title: Mutation testing
status: Accepted
scope: []
enforcement: process
---

# 0020: Mutation testing

## Decision

[go-mutesting](https://github.com/jonbaldie/go-mutesting), scoped to the diff of what's about to be committed via its git-diff mode, runs before every commit. A survived mutant means the code that changed wasn't actually pinned down by a test.

## Rationale

The mechanical version of `docs/development-practice.md`'s opening line: code shouldn't exist unless a test demanded it.

## Consequences

When a mutant survives, either the missing test gets written, or - if there's genuinely no behaviour worth testing - the code itself gets deleted rather than left with a gap next to it. go-mutesting is mid-transition to a successor project (`quality-gates/mutago`) - the tool name here may need to change later, the practice won't.

## Enforcement

This ADR *is* an enforcement mechanism for other decisions (`docs/adr/0013-implement-only-the-current-test.md`, `docs/adr/0014-outside-in-tdd.md`) rather than something checked by a subagent itself - it's a gate, not a shape to review.
