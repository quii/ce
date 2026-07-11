---
id: 0021
title: No flaky tests
status: Accepted
scope: []
enforcement: process
---

# 0021: No flaky tests

## Decision

A test that sometimes fails for no code reason is not tolerated, ever - no retries, no `@flaky`/skip annotations, no quarantine lists. The standard test invocation is `go test -race -count=3 -shuffle=on ./...`.

## Rationale

We control the entire environment, down to the Docker base image, and computers are deterministic - there's no such thing as an inherently flaky test on a machine we fully control, only a test with an unhandled source of non-determinism. Usual suspects:

- wall-clock time or `time.Sleep` used for synchronisation instead of the injected clock (`docs/adr/0015-utc-always.md`) or a proper signal (channel, waitgroup)
- a goroutine race - caught by `-race`, run as standard rather than as an occasional check
- shared mutable state between tests running in parallel (`t.Parallel()`)
- a test that depends on another test having already run - package-level state left behind, an assumed ordering. Every test must be independently executable and pass regardless of what ran before it or what order tests run in - this is exactly what `-shuffle=on` catches
- a testcontainers-backed test proceeding before the container is actually ready, instead of using a real readiness check
- reliance on map iteration order, or on wall-clock-derived values for uniqueness

## Consequences

Root causes get fixed at the source. Retrying, increasing a timeout, adding a sleep, or skipping a test are not fixes.

## Enforcement

Mechanical, via `-race`/`-count`/`-shuffle` plus a `forbidigo` rule denying `time.Sleep` outright (`.golangci.yml`) - not a subagent judgment call, a gate that either catches the flakiness or doesn't. The rule ended up global rather than scoped to `_test.go` files as originally planned: golangci-lint v2's `forbidigo` schema doesn't support a per-rule path, and production code reaching for raw `time.Sleep` instead of the injected `Clock`/a ticker isn't something worth allowing anyway - a stricter outcome than intended, kept deliberately rather than worked around.
