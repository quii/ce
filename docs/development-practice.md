# Development practice

This project follows the approach laid out in [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests) - not "we write tests," but tests as the thing that drives the design. Code should not exist unless a test demanded it.

See `docs/story-process.md` for where the first test in a story comes from, and for when to read the ADRs below - most stories drive a specification against a use-case in-port before anything below it gets written.

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
