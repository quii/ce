---
id: 0017
title: No comments that explain what the code does
status: Accepted
scope:
  - "**/*.go"
enforcement: judgment
---

# 0017: No comments that explain what the code does

## Decision

No comments that explain what the code does. Well-named identifiers already do that; a comment earns its place only when it captures a non-obvious *why* - a constraint, a workaround, a decision that would otherwise look arbitrary.

## Rationale

Over-commenting - explaining basic syntax, narrating the obvious - is a well-documented LLM anti-pattern that adds noise and cognitive load without adding information.

## Consequences

Code will look sparser than what narrative-comment habits produce. That's intended, not a gap.

## Enforcement

Judgment - a subagent reviewing new comments checks whether each one states a *why* or is a redundant paraphrase of the line(s) it sits next to.
