---
id: 0010
title: Tiny types, not stringly typed code
status: Accepted
scope:
  - internal/domain/**
  - internal/ports/**
enforcement: judgment
---

# 0010: Tiny types, not stringly typed code

## Decision

Anything that isn't genuinely "just a string" gets its own named type - `type ThreadID string`, `type ParticipantID string`, `type MessageText string` - rather than passing `string` around everywhere and trusting call sites not to mix two of them up. The tiny type is also the natural home for whatever validation or parsing that value needs - `func NewThreadID(raw string) (ThreadID, error)`.

This deliberately does not extend to a use case's `Command` struct fields (`docs/adr/0003-commands-not-parameter-lists.md`). A `Command` represents raw, not-yet-validated input crossing into the application - it stays plain primitives, and the tiny type gets constructed once, inside the use case itself, as the first step of turning that raw command into something the domain can operate on.

## Rationale

The compiler doesn't know a `ThreadID` and a `ParticipantID` are "different kinds of string" unless it's told; once it is, it will stop one being passed where the other is expected, at compile time, for free. Validation is meant to live once, rather than being re-checked - or quietly forgotten - at every call site.

Putting the tiny type directly on a `Command` field looks like it achieves that, but doesn't: a defined type like `type ThreadID string` can still be constructed directly (`ThreadID(raw)`), bypassing its constructor entirely, so nothing stops some caller from skipping validation regardless of where the field's type says it "should" happen. Constructing the tiny type once, inside the use case, from a plain `Command` field is what actually guarantees it - every caller funnels through the same use case method no matter which adapter built the `Command`, so there's exactly one place validation can happen, not one place it's merely supposed to.

## Consequences

More types to define than a stringly-typed version would have, deliberately. `Command` fields are the one deliberate exception to this ADR's own rule - primitive by design, not an oversight.

## Enforcement

Judgment - no generic linter can determine "this string should be a named type." A subagent reviewing new fields or parameters in domain-adjacent code checks for primitive obsession.
