---
id: 0007
title: Thin HTTP handlers
status: Accepted
scope:
  - internal/adapters/httpapi/**
enforcement: judgment
---

# 0007: Thin HTTP handlers

## Decision

A handler's job: translate a generated request object into a command, call the relevant use case (in port), translate the result into a generated response object. Nothing else. If a handler needs an `if` that isn't about that translation, that logic belongs in the use case, not the handler.

Since `docs/adr/0024-openapi-spec-first-with-oapi-codegen.md`, wire-level concerns - request parsing, status codes, content negotiation, JSON (de)serialisation - live entirely in generated code (`*.gen.go`), which the handler never touches. The handler only ever sees the generated strict-server request/response types on one side and the in-port's `Command`/domain types on the other.

## Rationale

Keeping handlers this dumb is what makes the in-ports properly testable in isolation - the first test for a story drives a use case directly, with no HTTP layer involved at all (`docs/adr/0014-outside-in-tdd.md`).

## Consequences

Any decision beyond translation has to live somewhere else - a handler is never the place business logic accumulates, even a little.

## Enforcement

Judgment - there's no static check for "business logic leaked into a handler." A subagent reviewing a changed handler checks whether it does anything beyond parse/call/translate.
