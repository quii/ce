---
id: 0022
title: Specifications and drivers for acceptance-level tests
status: Accepted
scope:
  - specifications/**
enforcement: judgment
---

# 0022: Specifications and drivers for acceptance-level tests

## Decision

Acceptance-level tests are written as **specifications** - domain-centric, implementation-agnostic descriptions of behaviour, coupled to a `Driver` interface rather than to any specific environment. A **driver** is an adapter that translates a specification's calls into interaction with a particular deployment: an in-process driver calling the use-case in-port directly, an HTTP driver talking to a real running instance over the network. The same specification runs unchanged against every driver that implements it - the driver changes, the specification doesn't.

The in-process driver is what runs by default (fast, part of `go test -race -count=3 -shuffle=on ./...`). A container driver runs the same specifications against the real CE image, brought up via testcontainers-go the same way the Postgres contract tests already bring up real Postgres - not a separate shell-orchestrated suite outside `go test`.

See [Scaling Acceptance Tests](https://quii.gitbook.io/learn-go-with-tests/testing-fundamentals/scaling-acceptance-tests) for the full pattern this is drawn from.

## Rationale

Essential complexity (the domain behaviour being specified) and accidental complexity (networks, HTTP, containers) get separated cleanly: a specification only changes when the behaviour it describes changes, never because of how it's currently being run. Running the identical specification against the in-process driver and the container driver is what gives confidence the containerized deployment actually matches in-process behaviour, without maintaining a second, parallel E2E suite that could drift from the first the same way `docs/adr/0016-dont-loosen-a-test.md` guards against drift within a single suite.

## Consequences

Every specification is written against the `Driver` interface from the start, not against a concrete HTTP client or a concrete in-process call - retrofitting a driver abstraction onto a specification written against one environment specifically is exactly the "overly coupled to implementation" problem this pattern exists to avoid. The container driver is real infrastructure (a built image, a running container) - given the project is currently small, running it as part of every commit's gate is fine; if that stops being true, the cadence gets revisited then, not preemptively.

## Enforcement

Judgment - a subagent reviewing a new specification checks whether it's expressed against the `Driver` interface (verbs like "post a message," not HTTP-specific or in-process-specific detail), and whether a driver implementation leaks specification-level assumptions the other driver can't satisfy.
