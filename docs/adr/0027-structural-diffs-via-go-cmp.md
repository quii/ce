---
id: 0027
title: Structural test diffs via go-cmp, no assertion library
status: Accepted
scope:
  - "**/*_test.go"
  - internal/assert/**
enforcement: judgment
---

# 0027: Structural test diffs via go-cmp, no assertion library

## Decision

Every test assertion - scalar or structural - goes through a small, hand-written generics-based helper package at `internal/assert`, not a third-party assertion library (`testify`, `matryer/is`, or one of the newer generics-based ones such as `alecthomas/assert`, `shoenig/test`, `peterldowns/testy`) and not an inline `if got != want { t.Errorf(...) }` either. The package's surface is deliberately small:

- `assert.Equal[T any](t, got, want T, context string, args ...any)` - structural equality for any value, via [`google/go-cmp`](https://github.com/google/go-cmp)'s `cmp.Diff`. One helper covers scalars, structs, and slices alike, rather than a struct/slice path through `cmp.Diff` and a separate scalar path through a bare `!=` - a single call site to learn and grep for.
- `assert.True`/`assert.False(t, got bool, context string, args ...any)` - boolean checks.
- `assert.NoErr(t, err error, context string, args ...any)` - fails and halts the test (`t.Fatalf`) on an unexpected error, since nothing else in the test can proceed meaningfully once a call that was supposed to succeed didn't.
- `assert.ErrorIs(t, err, target error, context string, args ...any)` - wraps `errors.Is`, for asserting a specific sentinel was returned.
- `assert.ErrorAs[T error](t, err error, context string, args ...any) T` - wraps `errors.As`, returning the extracted error so the caller can assert further on it (its message, say). Halts the test on failure to extract, like `NoErr` - a zero-value `T` isn't a meaningful thing to keep asserting against.
- `assert.Len[T any](t, got []T, want int, context string, args ...any)` - fails and halts the test if `len(got) != want`, same rationale as `NoErr`: a caller almost always indexes into `got` (`got[0]`) immediately after, which would panic on a shorter-than-expected slice if this were a non-halting check.
- `assert.Contains[T comparable](t, haystack []T, want T, context string, args ...any)` - slice membership, `comparable` being exactly the constraint Go's generics were built to express for this.

`context` (plus optional `args`) is formatted the same way `t.Errorf`'s format string is - it names the operation or field under test (`"GetConversation(%q)"`, `"Thread.Recipients"`), which every helper prefixes onto its failure message so a diff or boolean mismatch is never reported without saying what produced it (`docs/adr/0012-clear-assertion-messages.md`'s bar).

## Rationale

Every "modern, generics-based" assertion library surveyed is, underneath, a thin wrapper around `go-cmp` for legible diffs plus its own opinions on API surface (`require`-style fluent chains, custom comparator functions, an `EqualFunc` interface, and so on). The one genuine standard-library gap is structural diffing itself - stdlib's best offering is a `%#v` dump, unreadable for anything beyond a couple of fields. `go-cmp` closes that gap directly; everything past it in a full assertion library is bookkeeping this project can write itself in a few lines, matching the "no frameworks whatsoever" tech-stack choice and `docs/adr/0005-no-new-dependencies.md`'s bar - a dependency earns its place only when there's no legitimate stdlib way to do the job, and stays additive rather than reshaping how code is written.

An earlier version of this ADR carved scalar comparisons out of that decision - "there's no capability gap there, so no helper is needed for those" - and left every test to keep writing `if got != want { t.Errorf(...) }` by hand. In practice that meant every call site re-decided, case by case, whether a value was "simple enough" for a bare `!=` or needed `cmp.Diff`, and re-wrote the same "state operation, got, want" message shape from scratch every time (`docs/adr/0012-clear-assertion-messages.md`'s actual bar) with no shared enforcement that it had. `cmp.Diff` handles scalars exactly as well as structs - there's no reason `assert.Equal[T any]` needs to special-case them - so folding scalar comparisons into the one helper removes a distinction that was never load-bearing and gives every assertion, not just the structural ones, a consistent message shape for free. `comparable` is the one place a second constraint earns its keep: `Contains` needs `==`, which `cmp.Diff`'s reflection-based approach doesn't require and `any` doesn't guarantee.

## Consequences

**Exception to `docs/adr/0005-no-new-dependencies.md`**: `github.com/google/go-cmp` moves from an indirect dependency (already present transitively via existing tooling) to a direct `require`, added to the `internal/depcheck` allowlist - same treatment as `oapi-codegen/runtime`, `testcontainers-go`, `pgx/v5`, and `goose/v3` before it. It's already being pulled into the module graph today, so this doesn't introduce a new, previously-unvetted dependency - it makes an existing one direct and intentional.

`internal/assert` is a small, hand-written package, not a reimplementation of testify's API surface. Its eight helpers exist because a real, recurring comparison shape in this codebase's tests needed them.

That said, "implement only what the current test requires" (`docs/adr/0013-implement-only-the-current-test.md`) is a hard rule for production code, not for a small internal test-support library like this one - the bar for a new `internal/assert` helper is judgment about whether the shape is genuinely general-purpose for test assertions, not "does a call site exist today." `assert.Contains` was added on exactly that judgment before it had a real call site beyond its own unit test; it has one now (`specifications/conversation_projection.go`'s `assertRecipients` - `domain.Recipients` is a set, so membership per element is the correct check, not positional equality against a full expected slice).

Every assertion still has to carry enough context to act on the failure without reading the test body (ADR 0012) - a bare `cmp.Diff` output naming only field-level differences, with no indication of which operation or command produced it, doesn't clear that bar on its own; every `internal/assert` helper takes a `context` parameter for exactly this reason, and folds it into the failure message rather than leaving it to the diff alone.

`assert.Equal[T any]` isn't a universal replacement for every comparison: `cmp.Diff` panics on a struct with unexported fields it has no way to compare, unless the type has a public `Equal` method `cmp` recognises (`time.Time` has one; `time.Location` doesn't, which is exactly the trap `*time.Location` equality checks fall into - a plain `if got != want` is still the right, and only, tool there). This is a pre-existing `go-cmp` characteristic, not something `internal/assert` works around; a call site that hits it keeps its bare comparison rather than forcing one through `Equal`.

`internal/assert`'s helpers take more than two parameters (`t`, a value or two, `context`, `args ...any`), which is the shape this codebase's `argument-limit` revive rule would otherwise flag (`docs/adr/0003-commands-not-parameter-lists.md`'s target is production in-port signatures, not test-assertion helpers) - `internal/assert` is added to that rule's existing path-based exclusion list in `.golangci.yml`, the same mechanism already used for `internal/domain`, `internal/adapters`, and the others.

## Enforcement

Judgment - a subagent reviewing a new or changed test assertion checks whether it uses one of `internal/assert`'s helpers rather than an inline `if got != want`/`!reflect.DeepEqual`/`%#v` dump, or a new third-party assertion library.
