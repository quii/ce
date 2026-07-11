# Iteration 0 - scaffolding

This isn't a story in the sense `docs/story-process.md` describes - there's no user doing a job, so there's no example map, no Gherkin, no rules to converge on. It's the checklist for making every "enforced by X" claim in `docs/` actually true, before the first real story starts. Once everything here is checked off, this file itself is done; it doesn't move to `stories/completed/` the way a real story would, since there's no test reference to point at - it's the thing that makes test references possible.

## Repo skeleton

- [x] `go.mod` - module path `github.com/quii/ce`, Go 1.25
- [x] Directory layout matching `docs/architecture.md`: `internal/domain`, `internal/ports/{in,out}`, `internal/adapters/{memory,httpapi}`, `specifications/` (`docs/adr/0022-specifications-and-drivers.md`), and `cmd/ce` for the single image that runs as either the API role or the relay role (`docs/brief.md`, `docs/source-control.md`)
- [x] A literal hello-world slice - `GetGreetingUseCase` in-port, `GreetingHandler` HTTP handler, `memory.GreetingFinder` out-port implementation, exercised by one real specification through both drivers - proves the skeleton, the TDD loop, and the fakes pattern all actually work end to end

## Build tooling

- [x] `mage` set up (tracked via `go.mod`'s `tool` directive, not `require`) as the build runner - Go functions in `magefile.go`, not a shell/YAML DSL
- [x] `mage test` - `go test -race -count=3 -shuffle=on ./...`
- [x] `mage lint` - `golangci-lint run ./...`
- [x] `mage mutate` - `go-mutesting --git-diff-lines --git-diff-base=HEAD --fail-on-escaped --logger-agentic-json`
- [ ] Decide whether `docs/adr/0005-no-new-dependencies.md`'s allowlist check should also cover `go.mod`'s `tool` directive, or whether dev tooling is intentionally freer than runtime dependencies (still open)

## Linting - `.golangci.yml`

- [x] `testpackage` enabled
- [x] `depguard`, file-scoped to the domain package, denying `net`, `net/http`, `log`, `log/slog`, `database/sql`, `os`, and any import from this module other than the domain package itself - verified empirically (adding a `log` import to the domain package fails lint with the expected message, reverting it passes)
- [x] `forbidigo`, denying `^time\.Now$` and anything from `math/rand`/`crypto/rand` outside the injected `Clock`/ID-generator adapters
- [x] `forbidigo`, second rule denying `time\.Sleep` - **scope changed from "test files only" to everywhere**: golangci-lint v2's forbidigo schema doesn't support a per-rule `path`, so the rule is global. Production code shouldn't be reaching for raw `time.Sleep` over the Clock/ticker pattern anyway, so this is arguably a stricter, still-correct outcome - flagging the deviation from the original plan rather than silently accepting it
- [x] `revive`'s `argument-limit`, set to 2, scoped to `internal/ports/in`/`internal/ports/out` via an exclude-list of every other top-level package (RE2, which golangci-lint uses, has no negative lookahead, so "everywhere except X" has to be spelled out rather than expressed directly) - verified empirically (a 3-arg function in `ports/in` fails lint, the same function elsewhere wouldn't be scoped)
- [x] `revive`'s `file-length-limit`, set to 250, with an `exclude-rules` entry exempting `_test.go` files from that rule specifically (path *and* message matched, not a blanket revive exemption)

## Architecture test

- [x] A Go test (`internal/archtest`) that inspects the domain package's imports via the standard library's `go/build` (no third-party dependency needed) and fails if it imports anything from this module other than itself, including an explicit check for `log`/`log/slog`

## Dependency allowlist

- [x] A test (`internal/depcheck`) that parses `go.mod` directly (no third-party mod-parsing library) and fails if it contains any *direct* `require` beyond an explicit allowlist - verified empirically (an unapproved `require` line fails the test, reverting it passes). Deliberately only checks direct requires, not the full transitive closure `go list -m all` would show - testcontainers-go alone pulls in dozens of indirect dependencies, and policing those isn't what this ADR is for

## Mutation testing

- [x] `go-mutesting` installed, `--git-diff-lines --git-diff-base=HEAD` scopes it to the pending diff rather than the whole repo
- [x] `--logger-agentic-json` output wired up
- **Known limitation, discovered empirically**: `--git-diff-lines` doesn't find mutations in files that are entirely new and untracked by git - it needs the file to at least be `git add`-ed (staged) to see a diff against `HEAD`, and even then a brand-new file's mutation targeting seems unreliable (one file in a new package failed to compile in isolation during a mutation run). Verified working correctly for genuine *modifications* to already-committed files, which is what real future commits will mostly look like - this bulk scaffolding commit is the unusual case, not the common one

## Docker

- [x] `Dockerfile` - single image, both roles, multi-stage build into `gcr.io/distroless/static-debian12`
- [x] `docker-compose.yml` - Postgres, plus the image started twice (API role, relay role)
- [x] `docker compose up` actually brings up a working stack - verified with a real `curl` against the running API container. Caught a real bug in the process: the relay role's original `select {}` placeholder is an actual Go runtime deadlock (no goroutines, no channels - nothing could ever wake it), which crashed the container immediately. Fixed with a proper `os/signal` wait, which is also just the idiomatically correct way to keep a service alive and shut down gracefully

## Specifications and drivers

- [x] `Driver` interface defined for the hello-world specification
- [x] In-process driver implementation, calling the in-port directly
- [x] Container driver implementation, using testcontainers-go to build and run the real CE image (the same tool already managing Postgres for the contract tests, not separate shell orchestration) and talk to it over HTTP - verified: it actually builds the image, starts the container, waits for real HTTP readiness, and runs the same specification against it
- [x] The hello-world specification passes through both drivers as part of `mage test` - no separate, slower cadence for now; project's small enough that the cost isn't prohibitive yet (revisit if that changes)

## ADRs and the pre-commit ADR check

- [x] `docs/adr/` created, one file per decision from the mechanical/judgment/process classification, consistent template (status, scope/paths for relevance-filtering, enforcement type) - 22 ADRs
- [x] `docs/architecture.md`, `docs/standards.md`, `docs/development-practice.md` rewritten down to short pointers wherever their content moved into an ADR
- [x] `docs/story-process.md` updated so implementation starts by reading the path-relevant ADRs, after the story, before writing code
- [ ] The pre-commit hook itself: computes the diff, filters ADRs to those that are both path-relevant *and* judgment-tier (mechanically-enforced ADRs are skipped - `go test`/`golangci-lint`/mutation testing already gate those), spins up one subagent per remaining ADR against the diff
- [ ] Violation handling wired up as agreed: an obvious fix gets applied and retried by the coding agent; a non-obvious fix stops and starts a conversation rather than guessing

## Note on the first commit(s)

The commit that first introduces a gate can't be checked by that gate yet - e.g. the commit that adds `.golangci.yml` is the one making linting exist, it can't already have been linted by it. That's expected here, not a violation, and it's specific to this file - every commit after iteration 0 is expected to pass everything that exists by that point.
