# Architecture

We're loosely following a DDD / hexagonal shape. Dependencies point inward: adapters depend on ports, ports depend on the domain, and the domain depends on nothing.

## Ports and adapters

- **In ports** - use cases, the application's public surface (e.g. "start conversation", "post message", "edit message"). HTTP handlers call into these, nothing else.
- **Out ports** - interfaces for anything CE depends on externally, e.g. the event store, projections/read db. Adapters (Postgres, fakes) implement these interfaces; the domain and use cases only ever see the interface, never the concrete adapter.

## Domain purity

The domain package imports nothing else from this project - no ports, no infra, no framework. Only the standard library. If the domain needs something from outside itself, that need gets expressed as an out port, not a direct import.

This is what ports and adapters is actually for: it keeps business rules free of infrastructure concerns, and it means the domain package is exactly where anyone - or any agent - should look to find the rules the system enforces (see `docs/story-process.md`, "rules become code, not just prose").

One deliberate exception to "express it as an out port": logging. The domain never logs, not even through an injected logger out-port. Errors leave the domain as return values like anything else - it's whatever's outside (a use case, a handler) that decides whether and how to log them. Informational logs - "creating conversation," that kind of narrative - belong at the point where the command is created, not inside the domain that executes it. The domain's job is to decide what happened, not to narrate it.

Enforced by an architecture test, not just convention: a plain Go test, part of the normal `go test -race -count=3 -shuffle=on ./...` run, that loads the module's package graph and fails if the domain package imports anything from this module other than itself. `log` and `log/slog` get checked explicitly too, even though they're standard library - "only the standard library" isn't quite the same rule as "never logs," so the test can't just rely on the internal-imports check to catch this one. It's a real test that fails the build, not a linter warning that can be shrugged off.

## Rich domain, not anemic

Domain types get behaviour, not just fields, wherever they actually own an invariant. The canonical shape for an event-sourced aggregate: rehydrate it from its event history, call a method like `Thread.PostMessage(cmd PostMessageCommand) ([]Event, error)` that checks the rules against current state and returns the events to append (or an error) - nothing outside the aggregate is allowed to produce those events directly.

This is what makes `docs/story-process.md`'s "rules become code" actually hold: a story's rules become guard clauses inside that one method, in one discoverable place, rather than logic re-implemented (or half-forgotten) across whichever use-case functions happen to touch that aggregate. An anemic model - domain types as plain data, all the logic living in use-case code that operates on them from outside - gives up exactly that guarantee: nothing stops two different use cases from checking (or not checking) the same rule differently.

That said, richness is a consequence of a type owning a real invariant, not a target to hit uniformly. CE's own domain is deliberately thin - most of what a "rich" domain would normally model (the resource, participant identities, attachments) is explicitly not CE's concern (see `brief.md` non-goals). Where a type has no rule to enforce, it stays a plain struct; manufacturing behaviour for its own sake is the same mistake as an anemic model, just in the other direction.

## Thin HTTP handlers

A handler's job: parse the request into a command, call the relevant use case (in port), translate the result into a response. Nothing else. If a handler needs an `if` that isn't about an HTTP concern - status code, content negotiation, request parsing - that logic belongs in the use case, not the handler.

Unlike domain purity, this isn't mechanically enforced - there's no static check for "business logic leaked into a handler," so it's review discipline rather than a build-breaking test. Keeping handlers this dumb is also what makes the in-ports properly testable in isolation - see `docs/development-practice.md` on outside-in TDD, where the first test for a story drives a use case directly, with no HTTP layer involved at all.

## CQRS

Writes and reads take entirely separate paths. This isn't CQRS for its own sake, it falls out of the auditability requirement: writes append events, reads are served from read-optimised projections built from those events. See `docs/write-path.md` for how writes actually flow through the system.
