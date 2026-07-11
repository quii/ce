# Iteration 0 - scaffolding

This isn't a story in the sense `docs/story-process.md` describes - there's no user doing a job, so there's no example map, no Gherkin, no rules to converge on. It's the checklist for making every "enforced by X" claim in `docs/` actually true, before the first real story starts. Once everything here is checked off, this file itself is done; it doesn't move to `stories/completed/` the way a real story would, since there's no test reference to point at - it's the thing that makes test references possible.

## Repo skeleton

- [ ] `go.mod` - module path, Go version
- [ ] Directory layout matching `docs/architecture.md`: a domain package, ports (in/out), adapters (Postgres, fakes), a `specifications/` package (`docs/adr/0022-specifications-and-drivers.md`), and `cmd/` for the single image that runs as either the API role or the relay role (`docs/brief.md`, `docs/source-control.md`)
- [ ] A literal hello-world slice - one trivial in-port, one HTTP handler, one fake-backed out-port, exercised by one real specification - to prove the skeleton, the TDD loop, and the fakes pattern all actually work end to end before any real story leans on them

## Build tooling

- [ ] `mage` set up (tracked via `go.mod`'s `tool` directive, not `require`) as the build runner - Go functions in `magefile.go`, not a shell/YAML DSL
- [ ] `mage test` - `go test -race -count=3 -shuffle=on ./...`
- [ ] `mage lint` - `golangci-lint run ./...`
- [ ] `mage mutate` - `go-mutesting`, git-diff scoped
- [ ] Decide whether `docs/adr/0005-no-new-dependencies.md`'s allowlist check should also cover `go.mod`'s `tool` directive, or whether dev tooling is intentionally freer than runtime dependencies

## Linting - `.golangci.yml`

- [ ] `testpackage` enabled
- [ ] `depguard`, file-scoped to the domain package, denying `net`, `net/http`, `log`, `log/slog`, `database/sql`, `os`, and any import from this module other than the domain package itself
- [ ] `forbidigo`, denying `^time\.Now$` and anything from `math/rand`/`crypto/rand` outside the injected `Clock`/ID-generator adapters
- [ ] `forbidigo`, second rule, denying `time\.Sleep` inside `_test.go` files
- [ ] `revive`'s `argument-limit`, set to 2, scoped to the in-port/use-case package
- [ ] `revive`'s `file-length-limit`, set to 250, with an `exclude-rules` entry exempting `_test.go` files from that rule specifically (path *and* message matched, not a blanket revive exemption)

## Architecture test

- [ ] A Go test that loads the module's package graph and fails if the domain package imports anything from this module other than itself, including an explicit check for `log`/`log/slog` (stdlib, so the general import check alone wouldn't catch it)

## Dependency allowlist

- [ ] A test that fails if `go.mod` contains any `require` beyond an explicit allowlist (starts empty)

## Mutation testing

- [ ] `go-mutesting` installed, git-diff mode configured so it scopes to the pending diff rather than the whole repo
- [ ] `--logger-agentic-json` output wired up

## Docker

- [ ] `Dockerfile` - single image, both roles
- [ ] `docker-compose.yml` - Postgres, plus the image started twice (API role, relay role) per `docs/brief.md`'s developer-experience section
- [ ] `docker compose up` actually brings up a working stack, hello-world slice included

## Specifications and drivers

- [ ] `Driver` interface defined for the hello-world specification
- [ ] In-process driver implementation, calling the in-port directly
- [ ] Container driver implementation, using testcontainers-go to build and run the real CE image (the same tool already managing Postgres for the contract tests, not separate shell orchestration) and talk to it over HTTP
- [ ] The hello-world specification passes through both drivers as part of `mage test` - no separate, slower cadence for now; project's small enough that the cost isn't prohibitive yet (revisit if that changes)

## ADRs and the pre-commit ADR check

- [x] `docs/adr/` created, one file per decision from the mechanical/judgment/process classification, consistent template (status, scope/paths for relevance-filtering, enforcement type) - 22 ADRs
- [x] `docs/architecture.md`, `docs/standards.md`, `docs/development-practice.md` rewritten down to short pointers wherever their content moved into an ADR
- [x] `docs/story-process.md` updated so implementation starts by reading the path-relevant ADRs, after the story, before writing code
- [ ] The pre-commit hook itself: computes the diff, filters ADRs to those that are both path-relevant *and* judgment-tier (mechanically-enforced ADRs are skipped - `go test`/`golangci-lint`/mutation testing already gate those), spins up one subagent per remaining ADR against the diff
- [ ] Violation handling wired up as agreed: an obvious fix gets applied and retried by the coding agent; a non-obvious fix stops and starts a conversation rather than guessing

## Note on the first commit(s)

The commit that first introduces a gate can't be checked by that gate yet - e.g. the commit that adds `.golangci.yml` is the one making linting exist, it can't already have been linted by it. That's expected here, not a violation, and it's specific to this file - every commit after iteration 0 is expected to pass everything that exists by that point.
