# Greet a particular name

As an API caller, I can pass a name to `/greeting` and get a greeting personalized with it, instead of always getting the generic default.

## Rules

1. When a non-blank `name` query parameter is supplied, the greeting is `"Hello, <name>!"` using that value.
2. The name is trimmed of leading/trailing whitespace before being used.
3. When `name` is absent, empty, or whitespace-only, the greeting defaults to `"Hello, World!"`.
4. If `name` is repeated in the query string, only the first value is used.
5. No validation is applied to the name's content or length - anything (post-trim) is accepted and used as-is.

## Scenarios

```gherkin
Feature: Greet a particular name

  Scenario: A name is supplied
    Given no prior state
    When a caller requests a greeting with name "Chris"
    Then the greeting is "Hello, Chris!"

  Scenario: A name with surrounding whitespace is trimmed
    Given no prior state
    When a caller requests a greeting with name " Chris "
    Then the greeting is "Hello, Chris!"

  Scenario: No name is supplied
    Given no prior state
    When a caller requests a greeting with no name
    Then the greeting is "Hello, World!"

  Scenario: An empty name is treated as no name
    Given no prior state
    When a caller requests a greeting with name ""
    Then the greeting is "Hello, World!"

  Scenario: A whitespace-only name is treated as no name
    Given no prior state
    When a caller requests a greeting with name "  "
    Then the greeting is "Hello, World!"

  Scenario: A repeated name parameter uses the first value
    Given no prior state
    When a caller requests a greeting with name "Chris" and name "Sam"
    Then the greeting is "Hello, Chris!"

  Scenario: A name with unrestricted characters is used as-is
    Given no prior state
    When a caller requests a greeting with name "世界"
    Then the greeting is "Hello, 世界!"
```

## Specification

`specifications.GreetingSpecification` (`specifications/greeting.go`) exercises rules 1, 2, 3, and 5 (six scenarios), run via `TestGreeting` against both the in-process driver (`specifications/inprocess/driver_test.go`) and the container driver (`specifications/container/driver_test.go`). Per `docs/adr/0022-specifications-and-drivers.md`, the specification is written directly against the real in-port, `in.Greeter` (`internal/ports/in/get_greeting.go`) - there's no bespoke `Driver` type. The in-process driver *is* the real use case, called directly; the container driver implements `Greeter` by translating a `GetGreetingCommand` into an HTTP request and the JSON response back into a `domain.Greeting`.

Rule 4 (a repeated `name` parameter uses the first value) is deliberately not part of the driver-agnostic specification: "repeated query key" is an HTTP transport detail with no equivalent below the HTTP layer, so the in-process driver couldn't exercise it against real production code - it could only fabricate the behaviour in driver glue. It's covered instead by a narrow unit test directly on the handler: `TestGreetingHandler_RepeatedNameParameterUsesFirstValue` (`internal/adapters/httpapi/greeting_handler_test.go`).

The rules are enforced in code as follows:

- Rule 1 & 5: `domain.Name.Greet()` (`internal/domain/greeting.go`) builds `"Hello, <name>!"` from any non-blank name, with no further validation.
- Rule 2: `domain.NewName` trims the raw input - `GetGreetingCommand.Name` is a plain `string` (a `Command` carries raw, unvalidated input - `docs/adr/0010-tiny-types.md`), and `GetGreetingUseCase.Greet` (`internal/ports/in/get_greeting.go`) constructs the trimmed `domain.Name` as its first step, the one place this happens no matter which driver built the command.
- Rule 3: `GetGreetingUseCase.Greet` falls back to `out.GreetingFinder` when the trimmed name is blank.
- Rule 4: `net/url`'s `Query().Get("name")` in the HTTP handler (`internal/adapters/httpapi/greeting_handler.go`) resolves a repeated query parameter to its first value before it ever reaches the use case.
