---
id: 0012
title: Clear assertion messages
status: Accepted
scope:
  - "**/*_test.go"
enforcement: judgment
---

# 0012: Clear assertion messages

## Decision

Run the failing test and read its output before writing any code. Assertion messages state what was expected, what was actually received, and enough context to act on it without having to go and read the test body. `false was not equal to true` is not acceptable - it communicates nothing 99.9999% of the time.

## Rationale

A test's failure output is the whole point of writing the test first - it's supposed to tell you what to do next. A test whose failure message doesn't communicate the problem has failed at the one thing it exists to do. (See the anti-patterns section of [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests).)

## Consequences

Bare boolean assertions or unadorned `reflect.DeepEqual` comparisons aren't good enough on their own, even when they're technically correct.

## Enforcement

Judgment - a subagent reviewing new or changed test assertions checks whether the failure output would actually communicate the problem, not just whether the assertion is logically correct.
