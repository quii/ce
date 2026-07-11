---
id: 0011
title: Domain errors stay domain errors
status: Accepted
scope:
  - internal/domain/**
  - internal/adapters/**
enforcement: judgment
---

# 0011: Domain errors stay domain errors

## Decision

Errors returned from domain or use-case code are sentinel or typed errors defined in the domain, never `net/http` status codes, never a driver-specific error like `sql.ErrNoRows` leaking up from an adapter. Translating a domain error into an HTTP status code is the handler's job and the handler's alone. An adapter that gets `sql.ErrNoRows` back from Postgres translates it to whatever the out port's interface promises (e.g. a domain-level `ErrNotFound`) before it ever leaves the adapter.

## Rationale

The same thin-handler boundary (`docs/adr/0007-thin-http-handlers.md`), applied to the error path instead of the happy path.

## Consequences

Every adapter is responsible for translating every driver-specific error it can produce, at the point it crosses out of the adapter - not later, and not by the caller having to know about the driver.

## Enforcement

Mixed. The `net/http` half is already covered mechanically by `docs/adr/0001-domain-purity.md`'s `depguard` rule - the domain can't import `net/http` at all, so it structurally can't return an HTTP status code. The `sql.ErrNoRows`-leak half has no mechanical check: a subagent reviewing a changed adapter checks whether any driver-specific error can escape untranslated.
