---
id: 0025
title: Composition root - wiring stays at the edge
status: Accepted
scope:
  - cmd/**
  - internal/domain/**
  - internal/ports/**
  - internal/adapters/**
enforcement: judgment
---

# 0025: Composition root - wiring stays at the edge

## Decision

Only `cmd/**` constructs things. A `main.go` is the one and only place a concrete out-adapter, use case, or in-adapter gets built and wired to the next - `memory.NewEventStore()`, `in.NewStartConversationUseCase(...)`, `httpapi.NewConversationHandler(...)` and their equivalents are called from `cmd/**` and nowhere else. `internal/domain` and `internal/ports/**` never construct their own dependencies - they're always *given* them through a constructor parameter. If a handler or use case needs something it doesn't already have, that need gets threaded through from `main`, not `new`ed up locally because it's easier in the moment.

Every binary's composition root wires its dependencies bottom-up, through the same four stages, from the first out-port and use case it has - not deferred until a second one shows up:

1. **Bootstrap** - read config/env, nothing else. The only place that touches the messy outside-configuration world.
2. **`OutPorts`** - a struct with one exported field per out-port the binary needs, each field typed as the out-port interface and populated with the concrete adapter. `NewOutPorts(cfg)` is the one place all out-adapters get constructed, even when there's only one of them today. Downstream construction in the same file references the fields by name (`in.NewX(ports.Clock, ports.Events)`) rather than passing the whole bundle - so every use case's actual dependencies show up at its callsite, not just inside its own constructor.
3. **`Application`** - a concrete struct (not an interface - there's nothing to abstract over) bundling the constructed use cases, built from an `*OutPorts` value. Use plain, boring names for this - not "Hub" or similar.
4. **In-adaptors** - HTTP handlers, CLI, etc., built last, from `Application`.

Starting every binary on this shape from day one, rather than "flat until it hurts," is the point: there's no threshold to guess at, no moment where someone has to notice the fan-out has arrived and go refactor `main` under time pressure. The shape is already there waiting for the second out-port or use case.

## Rationale

An architecture diagram doesn't rot in the design - it rots in the wiring, in the small sins committed when a dependency gets constructed inline because threading it through felt like unnecessary ceremony at the time. Keeping construction confined to one place, in the same bottom-up order every time, means there's always exactly one place to look for how something gets built and exactly one place to add the next one - the same "screaming architecture" property `docs/adr/0007-thin-http-handlers.md` and `docs/adr/0001-domain-purity.md` already give the rest of the codebase, extended to cover the part those two ADRs don't: where objects come from in the first place.

## Consequences

`main.go` grows only along the bootstrap → `OutPorts` → `Application` → in-adaptors order as a binary's needs grow - never by adding a construction call somewhere else because it was more convenient there. A handler, a use case, or a domain type constructing a concrete dependency itself, rather than receiving it, is always a violation regardless of how small the dependency looks.

## History

Step 2 originally read "an interface aggregating every out-port the binary's use cases need," with the concrete adapters attached via embedded interface fields. That let a single-dependency use case take the aggregate directly - `in.NewGetConversationUseCase(ports)` picked up `out.Projection` implicitly through embedding - but it meant every multi-dependency callsite had to pass the same identifier several times (`in.NewManageThreadParticipantUseCase(ports, ports)`) with no indication at the callsite of which position was which port. Corrected once caught: the bundle is now a struct with named fields, so every callsite in the composition root reads as an inventory of what that specific use case actually needs, not a list of the same opaque identifier repeated. `NewApplication` takes `*OutPorts`, not the old interface type.

## Enforcement

Judgment, with one mechanical backstop:

- Mechanical: an architecture test (alongside `internal/archtest/domain_purity_test.go`) asserting that `internal/domain`, `internal/ports/in`, and `internal/ports/out` never import `internal/adapters/**`. This doesn't extend to "only `cmd/**` may reference an adapter package" - a driver in `specifications/**` legitimately depends on another adapter's generated client type; the rule this ADR actually needs mechanically checked is that the innermost layers (domain, ports) stay clean, not that no other package outside `cmd/**` ever names an adapter type.
- Judgment: a subagent reviewing a diff that touches a `cmd/**` binary's wiring checks that a new out-port lands inside that binary's `OutPorts`, a new use case inside its `Application`, and that no dependency was constructed somewhere other than the composition root.
