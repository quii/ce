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

Acceptance-level tests are written as **specifications** - domain-centric, implementation-agnostic descriptions of behaviour, coupled to an application's actual in-port interface rather than to a bespoke test-only abstraction. A **driver** is an implementation of that in-port interface for a particular deployment: the in-process driver *is* the real use case, called directly; a container driver translates the same domain-level call into interaction with a real running instance (an HTTP request/response, for a service exposed over HTTP). The same specification runs unchanged against every driver that implements the port - the driver changes, the specification doesn't.

For an HTTP-exposed service, the container driver's request/response translation is a thin wrapper around a generated client (`docs/adr/0024-openapi-spec-first-with-oapi-codegen.md`) rather than hand-rolled URL and JSON plumbing - the driver's own code is limited to mapping between the in-port's `Command`/domain types and the generated client's request/response types.

The in-process driver is what runs by default (fast, part of `go test -race -count=3 -shuffle=on ./...`) - it's just the production use case, no adapter code required. A container driver runs the same specifications against the real CE image, brought up via testcontainers-go the same way the Postgres contract tests already bring up real Postgres - not a separate shell-orchestrated suite outside `go test`.

See [Scaling Acceptance Tests](https://quii.gitbook.io/learn-go-with-tests/testing-fundamentals/scaling-acceptance-tests) for the full pattern this is drawn from.

## Rationale

A specification exercises the domain through its real in-port - the same `Command`/domain types production code uses, not a parallel primitive-only vocabulary invented for tests. This is consistent with how out-ports are already tested (`docs/adr/0009-contract-tests.md` runs contract tests against the real interface, not a stringly-typed stand-in) and reinforces the domain as a first-class citizen rather than something tests route around. Running the identical specification against the in-process driver and a container driver is what gives confidence the containerized deployment actually matches in-process behaviour, without maintaining a second, parallel E2E suite that could drift from the first the same way `docs/adr/0016-dont-loosen-a-test.md` guards against drift within a single suite.

Accidental complexity (networks, HTTP, containers) still stays out of the specification itself - it lives entirely inside a driver's translation to and from the in-port's `Command`, not in the specification's assertions.

## History

Originally, specifications were coupled to a bespoke `Driver` interface using plain primitives (`Greeting(ctx, name string) (string, error)`), deliberately kept separate from the in-port to avoid coupling a specification to internal types. That turned out to be the wrong trade-off: it duplicated the in-port's shape by hand for every driver, gave the in-process driver nothing to do but translate primitives into a `Command` (a whole file adding no value the real use case doesn't already provide), and couldn't naturally express "which in-port(s), exactly" for a story needing more than one. Corrected once caught: specifications now depend on the in-port interface directly. One flagged consequence: an HTTP-specific rule ("a repeated query parameter is rejected" - `stories/completed/greet-by-name.md`) can no longer be exercised by the in-process driver, since there's no multi-value concept below the HTTP layer once the in-port is the thing being called directly - that rule is covered by a focused unit test on the HTTP handler instead, not the shared specification.

## Consequences

Every specification is written against the relevant in-port interface(s) from the start, not against a concrete HTTP client or a concrete in-process call. A story needing more than one in-port composes them into a single interface for its driver to implement (ordinary Go interface embedding) rather than inventing a new bespoke type; in-port methods should be named for what they do (`Greet`, not `Handle`) so two embedded in-ports never collide on a shared generic method name. The container driver is real infrastructure (a built image, a running container) - given the project is currently small, running it as part of every commit's gate is fine; if that stops being true, the cadence gets revisited then, not preemptively.

## Enforcement

Judgment - a subagent reviewing a new specification checks whether it's expressed against the real in-port interface (verbs matching the in-port's own vocabulary, not HTTP-specific or in-process-specific detail), and whether a driver implementation leaks specification-level assumptions the other driver can't satisfy (e.g. an HTTP-only detail forced into the shared specification, or an in-port method named generically enough to collide when composed with another).
