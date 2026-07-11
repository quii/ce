---
id: 0006
title: Rich domain, not anemic
status: Accepted
scope:
  - internal/domain/**
enforcement: judgment
---

# 0006: Rich domain, not anemic

## Decision

Domain types get behaviour, not just fields, wherever they actually own an invariant. The canonical shape for an event-sourced aggregate: rehydrate it from its event history, call a method like `Thread.PostMessage(cmd PostMessageCommand) ([]Event, error)` that checks the rules against current state and returns the events to append (or an error) - nothing outside the aggregate is allowed to produce those events directly.

Richness is a consequence of a type owning a real invariant, not a target to hit uniformly. CE's own domain is deliberately thin (see `brief.md` non-goals); where a type has no rule to enforce, it stays a plain struct.

## Rationale

This is what makes "rules become code" (`docs/story-process.md`) actually hold: a story's rules become guard clauses inside one discoverable method, rather than logic re-implemented (or half-forgotten) across whichever use-case functions happen to touch that aggregate. An anemic model gives up exactly that guarantee.

## Consequences

Manufacturing behaviour on a type with no real invariant is the same mistake as an anemic model, just in the other direction - richness isn't the goal, correctly-placed invariant enforcement is.

## Enforcement

Judgment - a subagent reviewing a diff that touches the domain checks whether new behaviour sits inside the type that owns the invariant it's enforcing, versus having leaked into use-case code, or having been added to a type with no real rule to justify it.
