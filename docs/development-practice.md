# Development practice

This project follows the approach laid out in [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests) - not "we write tests," but tests as the thing that drives the design. Code should not exist unless a test demanded it.

See `docs/story-process.md` for where the first test in a story comes from, and for when to read the ADRs below - most stories drive a specification against a use-case in-port before anything below it gets written.

## Verifying HTTP-facing changes

Don't manually start the server (`go run ./cmd/api`) and `curl` it to check a change works - that's redundant with the test suite already in place, and each manual run/curl needs its own permission prompt for no real benefit. `specifications/container/driver_test.go` already exercises real HTTP behaviour against the actual compiled binary via testcontainers, and `specifications/inprocess/driver_test.go` covers the same behaviour without HTTP at all - `go tool mage test` running both is enough to know an HTTP-facing change works. If a change needs a driving test to prove it, write that test (`docs/adr/0022-specifications-and-drivers.md`, or a narrow handler-level test for transport-only detail) rather than checking by hand.

## Decisions

Specifics live in `docs/adr/`, each with its own rationale and enforcement mechanism:

- [0014 - Outside-in TDD](adr/0014-outside-in-tdd.md)
- [0013 - Implement only what the current failing test requires](adr/0013-implement-only-the-current-test.md)
- [0012 - Clear assertion messages](adr/0012-clear-assertion-messages.md)
- [0002 - Tests only exercise public APIs](adr/0002-tests-only-exercise-public-apis.md)
- [0008 - Fakes over mocks for out-ports](adr/0008-fakes-over-mocks.md)
- [0009 - Contract tests for fakes](adr/0009-contract-tests.md)
- [0022 - Specifications and drivers for acceptance-level tests](adr/0022-specifications-and-drivers.md)
- [0020 - Mutation testing](adr/0020-mutation-testing.md)
- [0021 - No flaky tests](adr/0021-no-flaky-tests.md)
