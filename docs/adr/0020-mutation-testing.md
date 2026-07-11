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

## Known tool quirks

- `--git-diff-lines` doesn't find mutations in files that are entirely new and untracked by git - the file needs to be `git add`-ed (staged) to see a diff against `HEAD` first (see `stories/iteration-0.md`).
- `specifications/greeting.go` reliably fails to compile in isolation (`undefined: Driver`) when go-mutesting builds it standalone, even against a completely empty diff (verified by running with `--git-diff-base=HEAD` on a clean tree matching `HEAD`). This is harmless: it prints to stderr but the run still reports `0 killed, 0 escaped` and exits `0`, so it doesn't block a commit. It's a pre-existing quirk in how the tool isolates that file, not something introduced by a particular change - no need to re-investigate it each time it shows up.
