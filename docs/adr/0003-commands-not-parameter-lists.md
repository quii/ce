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

An in-port method takes a single command struct, not a list of arguments. `PostMessage(cmd PostMessageCommand)`, not `PostMessage(threadID, author string, text string, attachments []string)`. The same applies to results coming back out of a use case where there's more than one thing to return alongside the error.

Scope: the **public in-port surface** - the interface itself and the exported methods on the use case struct that satisfy it. It doesn't extend to:

- **Constructors.** A use case takes its out-port dependencies directly as constructor arguments (`NewX(clock out.Clock, events out.EventStore)`), not bundled into a `XDependencies` struct. Interface stability isn't a concern here - the only caller is the composition root, which changes together with it - and a constructor's arg list growing to three, four, five ports is a real coupling signal worth seeing, not noise to hide behind a bundle.
- **Private helpers inside the use case implementation.** Those are ordinary Go: take whatever parameters make the code clearest, in whatever order. Bending an internal helper into a fake-command shape (`(ctx, cmd, action)` → `(ctx, op struct{Cmd, Action})`) trades real code clarity for zero real interface stability.

## Rationale

Interface stability as much as readability: adding a field to a struct doesn't break any existing caller, adding a parameter to a function signature does. Commands are data, not runtime configuration, so a plain struct is the right shape - not the functional-options pattern, which solves a different problem (optional, evolving configuration on a constructor).

Keeping the rule scoped to the public surface is deliberate. The whole reason to reach for a command struct is protecting a stable public contract - a private helper has no public contract to protect, and a constructor's contract is with exactly one collocated caller. Extending the rule beyond the public surface would just push both into a fake-command shape for no interface-stability win, hiding a coupling signal in the constructor case and hurting readability in the helper case.

## Consequences

Every in-port method has exactly one meaningful parameter besides `context.Context`. Constructors and private helpers are ordinary Go.

## Enforcement

Mechanical - `revive`'s `argument-limit` rule, set to 2 (room for `ctx` plus one command struct), scoped via `linters.exclusions.rules` in `.golangci.yml`:

1. Everything outside `internal/ports/**` is excluded outright.
2. Inside `internal/ports/**`, a `source`-regex exclusion (`^func (\([^)]+\) )?[a-z]|^func [A-Z]`) excludes every function declaration EXCEPT one that has a receiver AND an exported name - i.e. exactly `func (recv T) UpperName(...)`. That covers exported methods with a receiver (the in-port interface implementations) and leaves everything else - unexported methods, unexported free functions, and exported free functions (constructors) - out of scope.

The combination leaves the check biting exactly the public in-port surface - interface methods and the exported methods that satisfy them - and no other function in the codebase.
