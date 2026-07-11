---
id: 0001
title: Domain purity
status: Accepted
scope:
  - internal/domain/**
enforcement: mechanical
---

# 0001: Domain purity

## Decision

The domain package imports nothing else from this project - no ports, no infra, no framework, only the standard library. A need for something outside itself gets expressed as an out port, not a direct import.

One deliberate exception, not covered by "express it as an out port": logging. The domain never logs, not even through an injected logger out-port. Errors leave the domain as return values like anything else; informational, narrative logging belongs at the point where the command was created, not inside the domain that executes it.

## Rationale

This is what ports and adapters is actually for: it keeps business rules free of infrastructure concerns, and it means the domain package is exactly where anyone - or any agent - should look to find the rules the system enforces.

## Consequences

Any real external need (storage, IDs, time when it can't just ride on the command) must be expressed as an out port. Logging specifically never gets one, even though it might look harmless.

## Enforcement

Mechanical, two independent ways:

- An architecture test that loads the module's package graph and fails if the domain package imports anything from this module other than itself, including an explicit check for `log`/`log/slog` (standard library, so the general import check wouldn't otherwise catch it)
- `depguard`, file-scoped to the domain package, denying `net`, `net/http`, `log`, `log/slog`, `database/sql`, `os`, and any import from this module other than the domain package itself
