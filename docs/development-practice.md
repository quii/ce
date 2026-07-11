# Development practice

This project follows the approach laid out in [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests) - not "we write tests," but tests as the thing that drives the design. Code should not exist unless a test demanded it.

See `docs/story-process.md` for where the first test in a story comes from - most stories drive an acceptance test against a use-case in-port before anything below it gets written.

## Outside-in TDD

Start from the outside. The first test for a new piece of behaviour is at the boundary - typically an in-port/use-case test, sometimes an HTTP handler test - not a unit test for some internal type that doesn't have a caller yet.

Let the failing test tell you what needs to exist next. Don't design the internals up front and backfill tests afterwards - that produces tests that confirm the implementation rather than ones that would have caught a wrong design.

Red, green, refactor, in small steps: a failing test, the minimum code to pass it, then refactor with the test suite as the safety net. Implement only what the current failing test requires - don't build ahead of it in anticipation of where the story is probably going. If that means the next step is obvious, write the next test and take it, rather than writing the code for it early.

Run the failing test and read its output before writing any code. The output has to say what's actually wrong - `false was not equal to true` tells you nothing 99.9999% of the time, and a test whose failure message doesn't communicate the problem has failed at the one thing it exists to do. Assertion messages state what was expected, what was actually received, and enough context to act on it without having to go and read the test body.

## Tests only exercise public APIs

Every test file is in an external test package - `package mypkg_test`, never `package mypkg`. No exceptions.

This means a test can never reach into unexported internals. If something feels untestable without breaking that rule, the design or the public API needs to change - the test doesn't get special access to work around it.

Enforced by the `testpackage` linter rule (see `docs/standards.md`) - CI fails on any `_test.go` file that isn't in a `_test` package.

## Fakes over mocks for out-ports

See [Working without mocks](https://quii.gitbook.io/learn-go-with-tests/testing-fundamentals/working-without-mocks).

Every out-port (the event store, the outbox, projections - anything CE depends on externally) gets a real adapter and an in-memory fake, both satisfying the same interface. Domain and use-case code is tested against the fake: fast, no I/O, no test containers, no setup/teardown.

Mocks - test doubles that assert on which calls were made - are not used for out-ports. A fake is a real, working, simplified implementation. It lets a test express behaviour ("given this was stored, when I ask for it back, I get it") instead of implementation detail ("this method was called once with these arguments"), which means tests don't break when internals are refactored.

## Contract tests

A fake is only useful if it behaves the same as the real thing it stands in for. Each out-port has one shared contract test suite, written against the interface rather than any specific implementation, and it runs twice: once against the fake, once against the real adapter (e.g. real Postgres, brought up via testcontainers for the run). If the fake and the real adapter ever disagree, the contract test catches it.

There's no separate "CI tests" tier. `go test -race -count=3 -shuffle=on ./...` runs everything - fakes, contract tests, real dependencies via testcontainers included - and must give absolute confidence on its own. If it passes locally, it passes in CI; there's nothing CI checks that a developer (or an agent) can't check first.

## No flaky tests

A test that sometimes fails for no code reason is not tolerated, ever. This gets stated bluntly because a lot of what agents have seen in training data normalises it - retries baked into CI, `@flaky`/skip annotations, quarantine lists, "just re-run it." None of that is available here. If a test fails, something is genuinely wrong, in the code or in the test, and it gets fixed - not retried into passing.

The justification isn't discipline for its own sake: we control the entire environment, down to the Docker base image, and computers are deterministic. There's no such thing as an inherently flaky test on a machine we fully control - only a test with an unhandled source of non-determinism. Usual suspects:

- wall-clock time or `time.Sleep` used for synchronisation instead of the injected clock (see `docs/standards.md`) or a proper signal (channel, waitgroup)
- a goroutine race - run with `-race` as standard, not as an occasional check
- shared mutable state between tests running in parallel (`t.Parallel()`)
- a test that depends on another test having already run - package-level state left behind, an assumed ordering. Every test must be independently executable and pass regardless of what ran before it or what order tests run in.
- a testcontainers-backed test proceeding before the container is actually ready, instead of using a real readiness check
- reliance on map iteration order, or on wall-clock-derived values for uniqueness

Finding and removing the actual cause is the fix. Retrying, increasing a timeout, adding a sleep, or skipping the test are not fixes - they're the flakiness pretending to be resolved.

This is why the standard test invocation is `-count=3 -shuffle=on`, not just `-race` on its own: `-shuffle=on` randomises the order tests and subtests run in on every invocation (printing the seed on failure, so a failing order can be reproduced with `-shuffle=<seed>`), and exists specifically to catch order-dependence and shared-state assumptions before they become a problem by accident. `-count=3` runs everything three times to surface anything that only fails intermittently. Both convert "we hope nothing's order-dependent or occasionally flaky" into something actually checked on every run, not assumed.
