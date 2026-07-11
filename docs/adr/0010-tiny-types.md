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

## Rationale

The compiler doesn't know a `ThreadID` and a `ParticipantID` are "different kinds of string" unless it's told; once it is, it will stop one being passed where the other is expected, at compile time, for free. Validation lives once, rather than being re-checked - or quietly forgotten - at every call site.

## Consequences

More types to define than a stringly-typed version would have, deliberately.

## Enforcement

Judgment - no generic linter can determine "this string should be a named type." A subagent reviewing new fields or parameters in domain-adjacent code checks for primitive obsession.
