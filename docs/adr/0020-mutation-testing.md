---
id: 0020
title: Mutation testing
status: Accepted
scope: []
enforcement: process
---

# 0020: Mutation testing

## Decision

[gremlins](https://github.com/go-gremlins/gremlins), scoped to the diff of what's about to be committed, runs before every commit. A surviving (`LIVED`) mutant means the code that changed wasn't actually pinned down by a test.

## Rationale

The mechanical version of `docs/development-practice.md`'s opening line: code shouldn't exist unless a test demanded it.

## Consequences

When a mutant survives, either the missing test gets written, or - if there's genuinely no behaviour worth testing - the code itself gets deleted rather than left with a gap next to it.

## History

Originally used a private fork, `github.com/jonbaldie/go-mutesting` (itself forked from `avito-tech/go-mutesting`, forked from the now-unmaintained `zimmski/go-mutesting`). That fork's type-checking pass relied on the pre-Go-modules `go/build`/`golang.org/x/tools/go/loader` APIs: `go/build.ImportDir` can't resolve a module-relative import path, falls back to `loader.Config.CreateFromFilenames` with only the single mutated file, and any package split across multiple files (e.g. `specifications/driver.go` + `specifications/greeting.go`) then fails to type-check (`undefined: Driver`) - producing false-positive escapes for any code only reachable through the specifications layer, i.e. most of this project's domain and use-case logic, given outside-in TDD is the primary approach here. Root cause confirmed by hand (see git history on this file for the full trace) before migrating to `gremlins`, a from-scratch, actively-maintained tool built on the module-aware `golang.org/x/tools` packages, which doesn't exhibit the same failure - verified against this repo's actual `specifications/` package before switching.

## Enforcement

This ADR *is* an enforcement mechanism for other decisions (`docs/adr/0013-implement-only-the-current-test.md`, `docs/adr/0014-outside-in-tdd.md`) rather than something checked by a subagent itself - it's a gate, not a shape to review.

## Known tool quirks

- Diffing against `HEAD` doesn't find mutations in files that are entirely new and untracked by git - the file needs to be `git add`-ed (staged) first (see `stories/iteration-0.md`). This is a `git diff HEAD` limitation, not specific to any one mutation tool.
- gremlins treats an *empty* diff as "test everything" rather than "nothing to do" (see its `internal/diff` package: "if the diff is empty, it returns true for all positions"). Left alone, a docs-only commit would run a full-repo scan and could fail on pre-existing, unrelated debt. `mage mutate` guards against this itself: it skips the gremlins run entirely when `git diff --quiet HEAD -- '*.go'` reports no pending Go changes.
- This project's tests exercise most domain/use-case code from a separate `specifications` package (outside-in TDD), so gremlins needs `--coverpkg=./...` to attribute that coverage correctly - without it, cross-package-only code is wrongly reported as "not covered."
- The container-driver specification (`specifications/container/driver.go`) spins up a real Docker container per mutant, which needs a generous `--timeout-coefficient` (`30` verified sufficient locally) or mutants there falsely report as timed out rather than resolving.
- **gremlins v0.6.0's `--threshold-efficacy`/`--threshold-mcover` flags are silently inert - do not rely on them.** Confirmed by hand: setting `--threshold-efficacy` to `1`, `99.9`, and `100` against the same run (70% actual efficacy) produced exit code `0` every time. Root cause traced into both dependencies: `viper`'s pflag-to-value switch (`viper.go`) has cases for `int`, `bool`, `stringSlice`, `float64Slice`, etc., but no case for a bare `float64` flag, so it falls through and returns the flag's raw string; gremlins' `configuration.Get[float64]` then does `viper.Get(k).(float64)`, a type assertion against that string which fails silently and defaults to `0` - so `et > 0` in its threshold check is never true, regardless of the configured value. `mage mutate` does not pass these flags at all; it enforces the gate itself by parsing `-o`'s JSON report and failing on `mutants_lived > 0`.
