---
id: 0003
title: Commands, not parameter lists
status: Accepted
scope:
  - internal/ports/**
enforcement: mechanical
---

# 0003: Commands, not parameter lists

## Decision

An in-port takes a single command struct, not a list of arguments. `PostMessage(cmd PostMessageCommand)`, not `PostMessage(threadID, author string, text string, attachments []string)`. The same applies to results coming back out of a use case where there's more than one thing to return alongside the error.

## Rationale

Interface stability as much as readability: adding a field to a struct doesn't break any existing caller, adding a parameter to a function signature does. Commands are data, not runtime configuration, so a plain struct is the right shape - not the functional-options pattern, which solves a different problem (optional, evolving configuration on a constructor).

## Consequences

Every in-port method has exactly one meaningful parameter besides `context.Context`.

## Enforcement

Mechanical - `revive`'s `argument-limit` rule, set to 2 (room for `ctx` plus one command struct), scoped to the in-port/use-case package.
