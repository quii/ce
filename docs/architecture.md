# Architecture

We're loosely following a DDD / hexagonal shape. Dependencies point inward: adapters depend on ports, ports depend on the domain, and the domain depends on nothing.

## Ports and adapters

- **In ports** - interfaces modelling a user doing a job, the application's public surface (e.g. "start conversation", "post message", "edit message"). HTTP handlers call into these, nothing else.
- **Use cases** - the concrete implementation of an in-port, typically constructed with fast in-memory out-ports for tests and real ones in production. Specifications (`docs/adr/0022-specifications-and-drivers.md`) drive a use case through its in-port, not through use-case-specific internals.
- **Out ports** - interfaces for anything CE depends on externally, e.g. the event store, projections/read db. Adapters (Postgres, fakes) implement these interfaces; the domain and use cases only ever see the interface, never the concrete adapter.

## Decisions

Specifics live in `docs/adr/`, each with its own rationale and enforcement mechanism:

- [0001 - Domain purity](adr/0001-domain-purity.md)
- [0006 - Rich domain, not anemic](adr/0006-rich-domain-not-anemic.md)
- [0007 - Thin HTTP handlers](adr/0007-thin-http-handlers.md)
- [0018 - CQRS](adr/0018-cqrs.md)
- [0019 - Event sourcing with a transactional outbox](adr/0019-event-sourcing-transactional-outbox.md) - mechanics in `docs/write-path.md`
