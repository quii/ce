---
id: 0024
title: Spec-first OpenAPI with oapi-codegen
status: Accepted
scope:
  - api/openapi.yaml
  - internal/adapters/httpapi/**
  - specifications/container/**
enforcement: judgment
---

# 0024: Spec-first OpenAPI with oapi-codegen

## Decision

`api/openapi.yaml` is the single source of truth for the HTTP API contract. [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) generates two things from it, both committed to the repo:

- `internal/adapters/httpapi/server.gen.go` - the **strict server** interface (`StartConversation(ctx, request) (response, error)` and one method per operation), generated for the standard-library `http.ServeMux`. The adapter (e.g. `conversation_handler.go`) implements this interface and nothing else - no request parsing, no response encoding, that's all in the generated layer.
- `internal/adapters/apiclient/client.gen.go` - a typed HTTP client, used by the container driver (`specifications/container/driver.go`) instead of hand-rolled URL construction and JSON decoding.

Both are regenerated with `go generate ./...` after editing the spec; each generated file's package carries a `//go:generate` directive.

## Rationale

The strict server signature keeps HTTP concerns (status codes, `Content-Type`, (de)serialisation) entirely out of the adapter, reinforcing `docs/adr/0007-thin-http-handlers.md` - there's no `ServeHTTP` left to accidentally grow business logic into. The generated client removes the hand-rolled plumbing `docs/adr/0022-specifications-and-drivers.md`'s container driver previously needed, and gives the future relay role (inter-service HTTP) a ready-made client for free.

## Consequences

**Exception to `docs/adr/0005-no-new-dependencies.md`**: the generated server and client both import `github.com/oapi-codegen/runtime` for OpenAPI-style query parameter binding/encoding - this is a real runtime dependency, not build-time tooling, and is the one accepted exception to the stdlib-only rule, added to the `internal/depcheck` allowlist. `oapi-codegen` itself is a `tool` directive (build-time only, same treatment as `mage`/`golangci-lint`).

**Stricter parameter validation than hand-rolled code.** The generated binder rejects a repeated scalar query parameter with `400 Bad Request` rather than silently taking the first value the way `url.Values.Get` used to. Accepted as more spec-correct: an ambiguous request is now rejected instead of guessed at.

**Nothing hand-written touches wire format.** Editing the contract means editing `api/openapi.yaml` and regenerating, not hand-editing a `*.gen.go` file (`DO NOT EDIT` banners aren't a formality - see `docs/adr/0013-implement-only-the-current-test.md`'s spirit of keeping generated and hand-written code visibly separate).

## Enforcement

Judgment - a subagent reviewing a change to a handler under `internal/adapters/httpapi/**` or to `specifications/container/driver.go` checks that the diff never edits a `*.gen.go` file directly, and that a new dependency import in that file traces back to `api/openapi.yaml` plus a regeneration, not a fresh manual `go get`. The `github.com/oapi-codegen/runtime` allowlist entry is enforced mechanically by `internal/depcheck/dependency_allowlist_test.go`.
