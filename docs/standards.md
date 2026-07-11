# Standards

Three blanket gates apply to every package in this project: linting (which also covers the file-length limit), mutation testing on whatever just changed, and the checks in the sections below. All are strict - not aspirational, not something to quietly override because a particular file is inconvenient.

## Linting

`golangci-lint run ./...` must be clean. The full enabled linter set lives in `.golangci.yml`, not duplicated here - that avoids the doc and the config drifting apart. A couple of rules are called out specifically because they're the enforcement mechanism for something documented elsewhere:

- `testpackage` - every test file is an external test package (`package mypkg_test`), see `docs/development-practice.md`
- `depguard`, file-scoped to the domain package, denying `net`, `net/http`, `log`, `log/slog`, `database/sql`, `os`, and any import from this module other than the domain package itself - a lint-level backstop alongside the architecture test described in `docs/architecture.md`, catching the same violation two independent ways
- `forbidigo`, denying specific calls - `^time\.Now$`, anything from `math/rand`/`crypto/rand` - outside the injected `Clock`/ID-generator adapters. `depguard` can only ban a package wholesale, and the domain still needs to import `time` for the `time.Time` type itself, so this is what actually catches the call rather than the whole package - see "Time and randomness" below
- `forbidigo` again, this time denying `time\.Sleep` inside `_test.go` files specifically - it's the single most common source of the sleep-as-synchronisation flakiness described in `docs/development-practice.md`'s "No flaky tests" section, so it doesn't get to just be a habit to avoid
- `revive`'s `argument-limit` rule, set low (2 - room for `ctx` plus one command struct) and scoped to the in-port/use-case package, as the enforcement mechanism behind "Commands, not parameter lists" below
- `revive`'s `file-length-limit` rule, capped at 250 lines - see below

Suppressing a finding with `//nolint` requires a comment explaining why. A bare `//nolint` is itself a lint failure.

## File length

Every `.go` file is capped at 250 lines, enforced by `revive`'s `file-length-limit` rule as part of `golangci-lint run ./...` - not a bespoke script, and not a separate tool from the rest of linting. Treat 250 as an ambitious starting point, not a permanent constant - if it proves unworkable in practice we'll revise it, but the default should be to split the file, not raise the number.

`_test.go` files are exempt, via a golangci-lint `exclude-rules` entry scoped to that rule specifically (matched on both path and the rule's own message, not just path) - the exemption is only for length, not a blanket pass on every other revive rule for test files. Table-driven tests with a lot of cases legitimately run long, and splitting a test file for its own sake doesn't buy anything a shorter production file split does.

## Mutation testing

[go-mutesting](https://github.com/jonbaldie/go-mutesting), scoped to the diff of what's about to be committed via its git-diff mode - not the whole repo. A full-repo mutation run is too slow to be a normal part of the workflow, but checking only what just changed is cheap enough to run before every commit (see `docs/source-control.md`).

A survived mutant means the code that changed wasn't actually pinned down by a test - the mechanical version of `docs/development-practice.md`'s opening line: code shouldn't exist unless a test demanded it. When one turns up, either write the test that would have caught it, or - if there's genuinely no behaviour there worth testing - treat that as a sign the code itself shouldn't exist, and delete it rather than leave the gap next to it.

Use the `--logger-agentic-json` output to close the gap: a stable ID, the diff, surrounding context, nearby test files, and a hint for the killing test, per survived mutant - built to be handed to an agent directly rather than requiring a human to read a mutation report.

One caveat: go-mutesting is mid-transition to a successor project (`quality-gates/mutago`). The tool name here may need to change later; the practice won't.

## Commands, not parameter lists

An in-port takes a single command struct, not a list of arguments. `PostMessage(cmd PostMessageCommand)`, not `PostMessage(threadID, author string, text string, attachments []string)`.

This is about interface stability as much as readability: adding a field to a struct doesn't break any existing caller, adding a parameter to a function signature does. Given commands are data - not runtime configuration - a plain struct is the right shape here, not the functional-options pattern (which exists to solve a different problem: optional, evolving configuration on a constructor, not a fixed data payload). The same applies to results coming back out of a use case where there's more than one thing to return alongside the error.

Enforced by `revive`'s `argument-limit` rule (see "Linting" above) rather than left as something to remember - a use-case method that grows a third parameter fails the build, not just a review comment.

## Tiny types, not stringly typed code

Anything that isn't genuinely "just a string" gets its own named type - `type ThreadID string`, `type ParticipantID string`, `type MessageText string` - rather than passing `string` around everywhere and trusting call sites not to mix two of them up. The compiler doesn't know a `ThreadID` and a `ParticipantID` are "different kinds of string" unless we tell it; once we do, it will stop a `ParticipantID` being passed where a `ThreadID` is expected, at compile time, for free.

It's not just naming. The tiny type is also the natural home for whatever validation or parsing that value needs - `func NewThreadID(raw string) (ThreadID, error)` is where "what makes a valid thread ID" is decided, once, rather than re-checked (or quietly forgotten) at every call site. Applies anywhere a primitive is standing in for something domain-specific - IDs first and foremost, but also things like message text if it turns out to have rules of its own.

## Time and randomness

Domain and use-case code never generates an ID/UUID directly - an ID generator is injected as an out port, the same as anything else CE depends on externally. Production wires up a real generator; tests use a fake that returns a fixed value.

Time is more often data than a dependency. Most of the time the domain doesn't need to ask "what time is it" - it needs to know when something happened, and that's something the command should carry, not something the domain should fetch for itself. `PostMessageCommand` carries an `OccurredAt time.Time` field, stamped by whoever builds the command - typically the HTTP handler, since "now" at request time is an adapter concern - and the domain just reads it like any other field.

This keeps the domain honestly pure for the common case: no injected `Clock` needed just to record when something happened, and it's still fully deterministic to test - construct the command with whatever time you want. Reserve an actual injected `Clock` for the rarer case where domain code needs to reason about "now" independently of the command's own timestamp (e.g. comparing against the current time mid-decision, not just recording when the request came in) - and even then it's worth double-checking whether that's really a domain concern before reaching for one. Where a clock genuinely is needed, it's injected the same way as anything else: production wires up the real clock, tests use a fake that returns a fixed value.

Every `time.Time` is UTC, always - at the boundary where it's created (the handler stamping `OccurredAt`, a real or fake `Clock`), not converted later. Two timestamps representing the same instant can still fail an equality check if one carries a UTC location and the other a local one - exactly the kind of unhandled non-determinism `docs/development-practice.md`'s "No flaky tests" section is about, just easy to miss because it looks like a formatting detail rather than a correctness one.

## Domain errors stay domain errors

Errors returned from domain or use-case code are sentinel or typed errors defined in the domain, never `net/http` status codes, never a driver-specific error like `sql.ErrNoRows` leaking up from an adapter. Translating a domain error into an HTTP status code is the handler's job and the handler's alone - it's the same "thin handler" boundary from `docs/architecture.md`, applied to the error path instead of the happy path. An adapter that gets `sql.ErrNoRows` back from Postgres translates it to whatever the out port's interface promises (e.g. a domain-level `ErrNotFound`) before it ever leaves the adapter.

## Guardrails for agentic work

A few rules exist specifically because they're easy mistakes for an agent to make in good faith, not because a human on this project would be likely to do them:

- **Don't loosen a test to make it pass.** If a test fails, either the code is wrong or the test's expectation was wrong - fix whichever one it actually is. Widening an assertion, deleting a case, or softening what's being checked just to get back to green is never acceptable, even as a temporary measure.
- **No comments that explain what the code does.** Well-named identifiers already do that; a comment earns its place only when it captures a non-obvious *why* - a constraint, a workaround, a decision that would otherwise look arbitrary.
- **No new third-party dependencies without flagging it first.** This project is deliberately standard-library-only (see `brief.md`) - reaching for a package to solve something the standard library already handles is a signal to stop and check, not to `go get` and move on. Enforced, not just requested: a test fails if `go.mod` contains any `require` beyond an explicit allowlist (which starts empty) - `go get`-ing something new breaks the build immediately rather than depending on anyone remembering to flag it.
