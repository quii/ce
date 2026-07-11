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
- **`mage mutate`'s results cannot currently be trusted and should not be used to block a commit.** `specifications/greeting.go` reliably fails to compile in isolation (`undefined: Driver`) when go-mutesting builds it standalone - this was previously (wrongly) recorded here as harmless because it produced `0 killed, 0 escaped` against an empty diff. It is **not** harmless: once there are real mutations to test (verified while building the `greet-by-name` story), the same isolation failure causes go-mutesting to report false-positive "escaped" mutants for any code only reachable through the specifications layer - i.e. most of this project's domain and use-case logic, given outside-in TDD is the primary approach here. Proven by hand: manually applying one of the reported "escaped" mutations (a negated condition in `internal/domain/greeting.go`) and running the plain `go test ./...` (not go-mutesting's isolated per-file build) failed 5/5 affected subtests across both drivers - the test suite genuinely catches it, so the escape verdict was a tool artifact, not a real gap. Root cause not yet fixed; needs investigation into why go-mutesting's isolation build for `specifications/greeting.go` fails (likely related to `specifications/` being a multi-file package - `driver.go` + `greeting.go` - and the isolation copying only the mutated file). Until fixed, treat any `mage mutate` escape report as unverified and manually confirm (as above) before trusting it.
