---
id: 0026
title: SQL spec-first with sqlc and goose
status: Accepted
scope:
  - internal/adapters/postgres/**
enforcement: judgment
---

# 0026: SQL spec-first with sqlc and goose

## Decision

Postgres-backed out-port adapters are generated from hand-written SQL, not a hand-rolled `database/sql` layer or an ORM. Schema changes are ordered migration files under `internal/adapters/postgres/migrations/*.sql`, written in [goose](https://github.com/pressly/goose)'s format (`-- +goose Up` / `-- +goose Down`) and applied via goose's Go library, not its CLI. [sqlc](https://github.com/sqlc-dev/sqlc) reads that same migrations directory as its schema source - there is no separately hand-maintained `schema.sql` that could drift out of sync with what's actually been applied - and generates typed Go (`*.sql.go`, committed to the repo) from named queries under `internal/adapters/postgres/queries/*.sql`, targeting `jackc/pgx/v5`. Regenerated with `go generate ./...` after editing a migration or a query file.

`cmd/api` applies any pending migrations on startup, guarded by a Postgres advisory lock so concurrent replicas starting at the same time don't race each other; `internal/adapters/postgres` exposes this as a function the composition root calls before the API starts serving traffic. The same function is what test setup uses too - the testcontainers-backed contract tests (`docs/adr/0009-contract-tests.md`) and the container driver both need a real schema on an ephemeral Postgres instance before anything runs against it.

Adapter code (e.g. a Postgres-backed implementation of `out.EventStore`) implements the out-port interface by calling into the sqlc-generated `Queries` type - it never issues SQL directly.

## Rationale

Mirrors `docs/adr/0024-openapi-spec-first-with-oapi-codegen.md`'s shape: a single hand-written source of truth (SQL here, OpenAPI there), codegen fills the mechanical translation layer, and hand-written code stays limited to the seam - an adapter satisfying the out-port interface - rather than string-building SQL or hand-mapping rows to structs. This keeps CE's "no frameworks whatsoever" stance intact: sqlc's output is plain functions/methods over rows, not a query builder or an ORM's object graph, so no new abstraction sits between an out-port and Postgres.

Pointing sqlc at the migrations directory instead of a hand-maintained `schema.sql` means there's exactly one place schema changes are recorded, and it's the one that's actually authoritative - what's really been applied to a database - rather than a parallel description of it that can silently go stale.

## Consequences

**Exception to `docs/adr/0005-no-new-dependencies.md`, same shape as oapi-codegen's**: `github.com/jackc/pgx/v5` and `github.com/pressly/goose/v3` are both real runtime dependencies - pgx because generated and hand-written adapter code both use it directly, goose because migrations are applied through its library API (from `cmd/api`'s startup in production, and from test setup against testcontainers) rather than shelled out to a CLI. Both added to the `internal/depcheck` allowlist. `sqlc` itself is a `tool` directive (build-time only, same treatment as `mage`/`golangci-lint`/`oapi-codegen`) and needs no allowlist entry.

**Migration application is coupled to `cmd/api`'s startup.** The API image needs the migrations directory bundled in, and every API replica's boot briefly contends for the advisory lock before serving traffic. Accepted as the simpler shape over a dedicated migrate role/init container, given the project's current scale - revisit if migration timing or lock contention ever becomes a real operational problem.

**Nothing hand-written touches generated files.** `*.sql.go` carries sqlc's own generated-file header; it's edited only by changing a migration or a query in `queries/*.sql` and regenerating - never by hand.

**Pre-release exception: migrations may be squashed while nothing has shipped.** The "never a hand-edited or retroactively-changed existing migration" rule exists to protect data that's already been applied against a real, running database - rewriting history under it would either desync goose's version tracking from reality or silently drop a transformation real rows still depend on. Neither risk exists before anything has been released: there is no real data anywhere, only test fixtures recreated from scratch by every test run. Until this project's first real release, an accumulated chain of migrations may be squashed into a single file describing the current schema directly, deleting the ones it replaces, same as any other pre-release cleanup - see "simplify event storage." This exception itself is retired the moment there's a real deployment with real data to lose.

**Contract tests still apply in full.** Per `docs/adr/0009-contract-tests.md`, the same shared contract-test suite runs against each out-port's fake and its Postgres-backed (sqlc-generated) adapter, brought up via testcontainers. A mismatch between the fake's behaviour and what the generated queries actually do against real Postgres is exactly what that test is for.

## Enforcement

Judgment - a subagent reviewing a change under `internal/adapters/postgres/` checks that: a schema change is a new goose-formatted migration file, never a hand-edited or retroactively-changed existing one, unless it's a pre-release squash per the Consequences section above; a new query is added via a `.sql` file plus regeneration rather than a hand-written SQL string in Go; generated files aren't hand-edited; and a new out-port implementation has a corresponding contract test run against both the fake and the real adapter.
