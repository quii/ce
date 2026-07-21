# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Conversation Engine (CE): a containerised Go service providing threaded conversations about arbitrary "resources" (URLs) as commodity infrastructure, sitting behind a caller's own auth. See `brief.md` for the full product brief. Every state change is event-sourced for auditability - see "Write path" below.

## Commands

**Always verify through `go tool mage <target>`. Never substitute a bare `go build`/`go vet`/`go test ./...`/`golangci-lint`/`gremlins` invocation for one of these** - the mage targets are the actual gate this project checks against (build tags, race/shuffle flags, diff-scoping for mutation testing), and they're already permission-allowlisted, so reaching for a bare command instead only adds an avoidable prompt for a weaker check:

- `go tool mage testunit` - fast tier, no Docker: `go test -race -count=3 -shuffle=on ./...`. Use this in the inner dev loop.
- `go tool mage test` - full tier: adds `-tags=integration`, bringing in the Postgres contract tests and the testcontainers-backed HTTP specifications. This is the pre-commit gate (also enforced by a git hook) and what CI-equivalent confidence means here - there is no separate CI.
- `go tool mage lint` - `golangci-lint run --build-tags=integration ./...`
- `go tool mage mutate` - mutation testing scoped to the pending diff (`git diff HEAD`); only pays the Docker cost when the diff touches `internal/adapters/postgres/**` or `specifications/container/**`

Single test, fast tier: `go test ./internal/domain/... -run TestName -v` (a one-off scoped run like this is fine - it's whole-suite verification that should go through mage). A test under a `//go:build integration` file (anything in `internal/adapters/postgres/`, `specifications/container/`) needs `-tags=integration` added to that same invocation or it won't compile in.

`docker compose up` (or `go tool mage up`, which additionally guards against port conflicts and is preferred) brings up the whole stack - Postgres, `api`, `relay`, `web` - built from one `Dockerfile` via a `SERVICE` build arg per role. Don't manually `go run ./cmd/api` + `curl` to check an HTTP change works - `go tool mage test` already exercises real HTTP behaviour through both drivers (`docs/development-practice.md`).

## Development workflow

1. **`/story`** - example-mapping conversation, writes to `stories/backlog/<name>.md` (`docs/story-process.md`). No code.
2. **`coder` agent** on that story - implements it end-to-end via outside-in TDD, runs the fast tier while iterating and the full tier before handing back, moves the story to `stories/completed/` once a real specification backs it. Never commits.
3. **`/precommit`** before every commit (also a git hook gate) - runs `test`/`lint`/`mutate`, then one `adr-checker` subagent per judgment-tier ADR whose `scope` overlaps the diff. An obvious violation gets fixed and re-checked; a non-obvious one stops for a conversation rather than a guess (`docs/source-control.md`).
4. Mid-development mutation survivors (outside `/precommit`) go to the **`mutation-gap-closer`** agent - it writes the killing test, or deletes the code if there's no real behaviour worth testing.
5. **`story-drift-checker`** runs advisory-only when a completed story's linked specification changes, flagging Gherkin that no longer matches what the spec verifies.

Trunk-based: no feature branches, small commits straight to `main`, each one clean enough to stand alone (`docs/source-control.md`).

## Architecture

Hexagonal, dependencies point inward: `internal/adapters` → `internal/ports` → `internal/domain`, with `internal/domain` depending on nothing (no `net/http`, no `database/sql`, no logging - enforced via `depguard`).

- **`internal/domain`** - the rich domain model: tiny types (not raw strings), errors as domain sentinels, all business rules. Pure, synchronous, no I/O.
- **`internal/ports/in`** - one interface per use case ("start a conversation", "reply to a thread"), each named for the job a caller is doing. HTTP handlers call these and nothing else - handlers stay a thin translate-request→command→translate-response layer.
- **`internal/ports/out`** - interfaces for everything external (event store, outbox, projection). The domain/use-cases only ever see the interface.
- **`internal/adapters`** - concrete implementations: `memory` (fakes, used in fast tests), `postgres` (real, sqlc + goose migrations), `httpapi` (oapi-codegen generated server + hand-written handlers), `apiclient` (generated client, used by `web` and the container-driver specs), `webui` (the htmx demo frontend - deliberately outside the story process and the ADR-check rigor bar; see `docs/source-control.md`), `contracttest` (shared test suites run against both the memory fake and the real Postgres adapter, so they can never silently disagree).
- **`cmd/{api,relay,web}`** - the three deployable roles, each its own thin composition root wiring adapters into use cases (nothing else constructs a dependency - see ADR 0025).

### Write path (event sourcing + CQRS)

Every state change is an event, never a row mutation (full audit trail; an edit/delete is a new event, not an UPDATE/DELETE). A write: validate → append event + outbox row in one transaction → respond `202 Accepted` with `Location: .../resource?after=<seq>`. A separate `relay` role (single active instance, no locking needed) drains the outbox and applies events to read-optimised projections asynchronously. A client polls the same `Location`: `202` until the projection's checkpoint reaches that sequence, `200` once it has. A plain `GET` with no `after` param is always an unconditional, immediate read - `200`/`404`, never `202`. Full mechanics and sequence diagrams: `docs/write-path.md`.

### Specifications and drivers

An in-port's acceptance-level test ("specification") is written once against the in-port interface itself, then run through multiple "drivers": `specifications/inprocess` (calls the use case directly, synchronous relay) and `specifications/container` (drives the real HTTP API against a real Postgres + relay via testcontainers, gated behind `//go:build integration`). Adding a use case means writing one specification, not one test per driver.

### The ADR system

`docs/adr/*.md` are the actual decision records; `docs/architecture.md`, `docs/standards.md`, `docs/development-practice.md`, `docs/source-control.md` are thin indexes into them. Each ADR's frontmatter carries a `scope` (glob patterns) and an `enforcement` tier:

- `mechanical` - caught by `test`/`lint` already; never re-checked by an agent.
- `judgment` - the bulk of them; an `adr-checker` subagent evaluates the diff against these when `scope` overlaps changed files, driven by `/precommit`.
- `process` - a practice (e.g. outside-in TDD, no flaky tests), not something a diff-checker evaluates.

When touching a path, check which ADRs' `scope` cover it before writing code, not just at commit time - `docs/story-process.md`'s "Reading ADRs before writing code" section.

## Testing conventions

- **`internal/assert`** - this project's own generics-based assertion helpers (`Equal`, `True`, `False`, `NoErr`, `ErrorIs`, `ErrorAs`, `Len`, `Contains`), backed by `google/go-cmp`. Use these instead of hand-rolled `if got != want { t.Errorf(...) }` or a third-party assertion library (`docs/adr/0027-structural-diffs-via-go-cmp.md`). Every helper's `context` argument is a `t.Errorf`-style format string naming the operation/field under test - always pass one specific enough to act on without reading the test body.
- Fakes over mocks for every out-port (`internal/adapters/memory`), driven through their real public API, never a mocking library.
