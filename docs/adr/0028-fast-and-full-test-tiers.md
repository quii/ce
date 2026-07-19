---
id: 0028
title: Fast and full test tiers, split by build tag
status: Accepted
scope:
  - magefile.go
  - internal/adapters/postgres/**
  - specifications/container/**
enforcement: judgment
---

# 0028: Fast and full test tiers, split by build tag

## Decision

Every test that needs Docker to run - the Postgres contract tests (`internal/adapters/postgres/store_test.go`, `timestamps_test.go`) and the container-driver specifications (`specifications/container/driver_test.go`) - carries `//go:build integration`. Nothing else does, including `specifications/container/driver_unit_test.go` (plain `httptest`, no Docker) and `specifications/inprocess/driver_test.go` (already Docker-free per `docs/adr/0022-specifications-and-drivers.md`).

Two magefile targets exist because of this:

- `go tool mage test` passes `-tags=integration` and remains the pre-commit gate `docs/source-control.md` describes - identical coverage to before this ADR, still the thing giving full confidence with no separate CI to catch what it misses.
- `go tool mage testunit` runs the same command without the tag - in-memory only, no containers started. This is what the inner dev loop (outside-in TDD, red-green-refactor) uses; `mage test` stays reserved for right before handing back or committing.

`go tool mage lint` passes `--build-tags=integration` so the tagged files stay linted rather than silently dropping out of analysis.

`go tool mage mutate` keeps `--integration` unconditionally (`docs/adr/0020-mutation-testing.md` - required for any domain/use-case mutant to be killable at all, unrelated to Docker). It adds gremlins' own `--tags=integration` only when the pending diff touches `internal/adapters/postgres/**` or `specifications/container/**`. Otherwise, gremlins' per-mutant `go test ./...` reruns simply don't compile the Docker-backed test files in, so a mutation run over domain/use-case/HTTP-handler code - the common case - never starts a container, while still being checked against every in-memory specification that actually covers it.

## Rationale

`stories/iteration-0.md` accepted running the container driver as part of every `go test ./...` "no separate, slower cadence for now; project's small enough that the cost isn't prohibitive yet (revisit if that changes)." It changed: the container-driver specification builds two Docker images and starts three containers per run, and mutation testing under `--integration` reruns the *whole module's* tests per surviving-candidate mutant - so every mutant, regardless of which file it's in, was paying that cost. That's expensive enough to discourage the tight outside-in TDD loop this project is built around, and directly increases the cost of a multi-agent workflow that runs `mage test` far more often than a human would.

The in-process driver (`docs/adr/0022-specifications-and-drivers.md`) already exercises the same specifications, the same production use cases, without touching Postgres or HTTP - it just doesn't run by default in isolation from the container-backed ones. A build tag is the standard Go mechanism for "this test needs infrastructure the rest don't" (the same shape `database/sql`, `net`, and most cloud SDKs use for their own integration suites), and it composes for free with gremlins' own `-t`/`--tags` flag and golangci-lint's `--build-tags`, rather than requiring a bespoke test-selection mechanism.

Manipulating `cmd/api`'s wiring via an environment variable to swap in memory adapters was considered and rejected: it would give a second, parallel way to build the application's dependency graph, in tension with `docs/adr/0025-composition-root.md` ("only `cmd/**` constructs things... one concrete constructor"), for no capability the in-process driver doesn't already provide.

## Consequences

A new test that needs Docker (a new out-adapter following the Postgres pattern, or a new container-driver specification) gets `//go:build integration` from the start, the same way a new out-port gets a contract test from the start (`docs/adr/0009-contract-tests.md`). Forgetting the tag doesn't break `mage test` (still tagged, still runs everything) but does mean `mage testunit` and untagged mutation runs silently stop giving that file's changes any coverage at all - so the tag is part of what "done" looks like for that kind of file, not an optional annotation.

`mage test`'s guarantee is unchanged: it is still, on its own, "full confidence, nothing extra needed for CI" (`docs/source-control.md`). Only the inner loop and mutation testing get a cheaper default; the final gate before a commit lands does not get weaker.

## Enforcement

Judgment - a subagent reviewing a diff that adds a test file under `internal/adapters/postgres/**` or `specifications/container/**` checks that it starts real infrastructure (a testcontainers call, a real network) and, if so, carries `//go:build integration`; and that a diff touching `magefile.go`'s `Test`/`TestUnit`/`Lint`/`Mutate` targets doesn't reintroduce a mismatch between what each command claims to cover and what it actually runs.
