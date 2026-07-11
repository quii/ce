---
id: 0019
title: Event sourcing with a transactional outbox for the write path
status: Accepted
scope: []
enforcement: process
---

# 0019: Event sourcing with a transactional outbox for the write path

## Decision

Every state change is captured as an event, never a row mutation, delivered to read-optimised projections via a transactional outbox and a single-active-instance relay. Writes respond `202 Accepted` with a `Location` carrying an `after=<sequence>` cursor; reads are eventually consistent with the event log by design.

## Rationale

What makes full audit retrieval possible, and gives a clean answer for messages being editable/deletable (an edit or delete is a new event, not an UPDATE/DELETE) - see `brief.md`'s auditability requirement.

## Consequences

The full mechanics - the write-path sequence, the api/relay role split, the polling/cursor semantics, both sequence diagrams - live in `docs/write-path.md`. This ADR is the decision record; that doc is the how.

## Enforcement

None directly at this ADR's level - see `docs/write-path.md` for the mechanical detail. Individual pieces of it (domain purity, no logging in the domain, UTC timestamps) are covered by their own ADRs.
